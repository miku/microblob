package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

// Version of application.
const Version = "0.1.4"

// Entry associates a key with a section in a file specified by offset and length.
type Entry struct {
	Key    string `json:"k"`
	Offset int64  `json:"o"`
	Length int64  `json:"l"`
}

// Backend abstracts various implementations.
type Backend interface {
	Get(key string) ([]byte, error)
	WriteEntries(entries []Entry) error
	Close() error
}

// KeyExtractor extracts a string key from data.
type KeyExtractor interface {
	ExtractKey([]byte) (string, error)
}

// KeyFunc extracts a key from a blob.
type KeyFunc func([]byte) (string, error)

// EntryWriter writes entries to some storage, e.g. a file or a database.
type EntryWriter func(entries []Entry) error

// LineProcessor reads a line, extracts the key and writes entries.
type LineProcessor struct {
	r         io.Reader   // input data
	f         KeyFunc     // extracts a string key from a byte blob
	w         EntryWriter // serializes entries
	BatchSize int         // number of lines in a batch
}

// Run starts processing the input, sequential version.
func (p LineProcessor) Run() error {
	bw := bufio.NewReader(p.r)
	for {
		b, err := bw.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		key, err := p.f(b)
		if err != nil {
			return err
		}
		entry := Entry{Key: key}
		if err := p.w([]Entry{entry}); err != nil {
			return err
		}
	}
	return nil
}

// workPackage is a unit of work handed to a worker.
type workPackage struct {
	docs   [][]byte // list of documents to work on
	offset int64    // offset to start with
}

// RunWithWorkers start processing the input, uses multiple workers.
func (p LineProcessor) RunWithWorkers() error {

	// workerErr is set, when a worker fails. Winds down processing.
	var processingErr error

	// Setup communication channels.
	work := make(chan workPackage)
	updates := make(chan []Entry)
	done := make(chan bool)

	// collector runs the EntryWriter on all incoming batches.
	collector := func(ch chan []Entry, done chan bool) {
		for batch := range ch {
			if err := p.w(batch); err != nil {
				processingErr = err
				break
			}
		}
		done <- true
	}

	// worker takes a workPackage, creates Entries from bytes and sends the result
	// down the sink.
	worker := func(queue chan workPackage, wg *sync.WaitGroup) {
		defer wg.Done()
		for pkg := range queue {
			offset := pkg.offset
			var entries []Entry
			for _, b := range pkg.docs {
				key, err := p.f(b)
				if err != nil {
					processingErr = err
					break
				}
				length := int64(len(b))
				entries = append(entries, Entry{key, offset, length})
				offset += length
			}

			updates <- entries
			if processingErr != nil {
				break
			}
		}
	}

	var wg sync.WaitGroup

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(work, &wg)
	}

	go collector(updates, done)

	br := bufio.NewReader(p.r)
	var offset, blen int64
	batch := [][]byte{}

	for {
		b, err := br.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(batch) == p.BatchSize {
			bb := make([][]byte, len(batch))
			copy(bb, batch)
			work <- workPackage{docs: bb, offset: offset}
			offset += blen
			blen, batch = 0, nil
		}
		batch = append(batch, b)
		blen += int64(len(b))
	}

	bb := make([][]byte, len(batch))
	copy(bb, batch)
	work <- workPackage{docs: bb, offset: offset}

	close(work)
	wg.Wait()
	close(updates)
	<-done

	return processingErr
}

// renderString tries various ways to get a string out of a given type.
func renderString(v interface{}) (s string, err error) {
	switch w := v.(type) {
	case string:
		s = w
	case int:
		s = fmt.Sprintf("%d", w)
	case float64:
		s = fmt.Sprintf("%0d", int(w))
	case fmt.Stringer:
		s = fmt.Sprintf("%s", w)
	case time.Time:
		s = w.Format(time.RFC3339)
	default:
		err = fmt.Errorf("unsupported type: %v", reflect.TypeOf(w))
	}
	return
}

// RegexpExtractor extract a key via regular expression.
type RegexpExtractor struct {
	Pattern *regexp.Regexp
}

// ExtractKey returns the key found in a byte slice.
func (e RegexpExtractor) ExtractKey(b []byte) (string, error) {
	return string(e.Pattern.Find(b)), nil
}

// ParsingExtractor parses JSON and extracts the top-level key at the given path.
type ParsingExtractor struct {
	Key string
}

// ExtractKey extracts the key.
func (e ParsingExtractor) ExtractKey(b []byte) (string, error) {
	dst := make(map[string]interface{})
	if err := json.Unmarshal(b, &dst); err != nil {
		return "", err
	}
	if _, ok := dst[e.Key]; !ok {
		return "", fmt.Errorf("key %s not found in: %s", e.Key, string(b))
	}
	s, err := renderString(dst[e.Key])
	if err != nil {
		return "", err
	}
	return s, nil
}

// leveldbBackend writes entries into leveldb.
type leveldbBackend struct {
	Blobfile string
	blob     *os.File
	Filename string
	db       *leveldb.DB
	sync.Mutex
}

func (b *leveldbBackend) openBlob() error {
	// TODO(miku): Store a SHA of the origin file in the blob store, compare with the
	// SHA of the currently used blob file.
	if b.blob != nil {
		return nil
	}
	file, err := os.Open(b.Blobfile)
	if err != nil {
		return err
	}
	b.blob = file
	return nil
}

func (b *leveldbBackend) openDatabase() error {
	if b.db != nil {
		return nil
	}
	db, err := leveldb.OpenFile(b.Filename, nil)
	if err != nil {
		return err
	}
	b.db = db
	return nil
}

func (b *leveldbBackend) Get(key string) ([]byte, error) {
	if err := b.openDatabase(); err != nil {
		return nil, err
	}

	value, err := b.db.Get([]byte(key), nil)
	if err != nil {
		return nil, err
	}

	obuf := bytes.NewBuffer(value[:8])
	lbuf := bytes.NewBuffer(value[8:])

	offset, err := binary.ReadVarint(obuf)
	if err != nil {
		return nil, err
	}

	length, err := binary.ReadVarint(lbuf)
	if err != nil {
		return nil, err
	}

	if err := b.openBlob(); err != nil {
		return nil, err
	}

	// Retrieve content.
	b.Lock()
	defer b.Unlock()
	if _, err := b.blob.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if _, err := b.blob.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (b *leveldbBackend) Close() error {
	if b.db != nil {
		if err := b.db.Close(); err != nil {
			return err
		}
	}
	if b.blob != nil {
		if err := b.blob.Close(); err != nil {
			return err
		}
	}
	return nil
}

// WriteEntries writes entries as batch into LevelDB. The value is fixed 16 byte
// slice, first 8 bytes represents the offset, last 8 bytes the length.
func (b *leveldbBackend) WriteEntries(entries []Entry) error {
	if err := b.openDatabase(); err != nil {
		return err
	}
	batch := new(leveldb.Batch)
	for _, entry := range entries {
		offset, length := make([]byte, 8), make([]byte, 8)
		binary.PutVarint(offset, entry.Offset)
		binary.PutVarint(length, entry.Length)
		value := append(offset, length...)
		batch.Put([]byte(entry.Key), value)
	}
	return b.db.Write(batch, nil)
}

func loggingWriter(entries []Entry) error {
	for _, e := range entries {
		fmt.Printf("%s\t%d\t%d\n", e.Key, e.Offset, e.Length)
	}
	return nil
}

// filterEmpty removes empty strings from a slice array.
func filterEmpty(ss []string) (filtered []string) {
	for _, s := range ss {
		if strings.TrimSpace(s) == "" {
			continue
		}
		filtered = append(filtered, s)
	}
	return
}

// BlobHandler serves blobs.
type BlobHandler struct {
	Backend Backend
}

func (h *BlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := filterEmpty(strings.Split(r.URL.Path, "/"))
	if len(parts) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`key is required`))
		return
	}
	key := strings.TrimSpace(parts[0])
	b, err := h.Backend.Get(key)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(b)
}

func main() {
	pattern := flag.String("r", "", "regular expression to use as key extractor")
	keypath := flag.String("key", "", "key to extract")
	dbname := flag.String("backend", "leveldb", "backend to use, currently only leveldb")
	dbfile := flag.String("db", "data.db", "filename to use for backend")
	blobfile := flag.String("file", "", "file to index or serve")
	serve := flag.Bool("serve", false, "serve file")
	addr := flag.String("addr", "127.0.0.1:8820", "address to serve")
	batchsize := flag.Int("batch", 100000, "number of lines in a batch")
	version := flag.Bool("version", false, "show version and exit")

	flag.Parse()

	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *blobfile == "" {
		log.Fatal("need a file to index or serve")
	}

	var backend Backend

	switch *dbname {
	default:
		backend = &leveldbBackend{Filename: *dbfile, Blobfile: *blobfile}
	case "tsv":
		log.Fatal("not a full backend yet")
	}
	defer backend.Close()

	// Serve content.
	if *serve {
		http.Handle("/", &BlobHandler{Backend: backend})
		log.Printf("serving blobs from %s on %s", *blobfile, *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			log.Fatal(err)
		}
	}

	// Index content.
	if *pattern == "" && *keypath == "" {
		log.Fatal("key or pattern required")
	}

	var extractor KeyExtractor

	if *pattern != "" {
		extractor = RegexpExtractor{Pattern: regexp.MustCompile(*pattern)}
	}
	if *keypath != "" {
		extractor = ParsingExtractor{Key: *keypath}
	}

	file, err := os.Open(*blobfile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	processor := LineProcessor{
		r:         file, // os.Stdin
		f:         extractor.ExtractKey,
		w:         backend.WriteEntries, // loggingWriter
		BatchSize: *batchsize,
	}
	if err := processor.RunWithWorkers(); err != nil {
		log.Fatal(err)
	}
}
