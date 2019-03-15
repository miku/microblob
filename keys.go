package microblob

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/schollz/progressbar"
	log "github.com/sirupsen/logrus"
)

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
	r                 io.Reader   // input data
	f                 KeyFunc     // extracts a string key from a byte blob
	w                 EntryWriter // serializes entries
	BatchSize         int         // number of lines in a batch
	InitialOffset     int64       // allow offsets beside zero
	Verbose           bool
	IgnoreMissingKeys bool // skip document with missing keys
}

// NewLineProcessor reads lines from the given reader, extracts the key with the
// given key function and writes entries to the given entry writer.
func NewLineProcessor(r io.Reader, w EntryWriter, f KeyFunc) LineProcessor {
	return NewLineProcessorBatchSize(r, w, f, 100000)
}

// NewLineProcessorBatchSize reads lines from the given reader, extracts the key with the
// given key function and writes entries to the given entry writer. Additionally,
// the number of lines per batch can be specified.
func NewLineProcessorBatchSize(r io.Reader, w EntryWriter, f KeyFunc, size int) LineProcessor {
	return LineProcessor{r: r, w: w, f: f, BatchSize: size}
}

// workPackage is a unit of work handed to a worker.
type workPackage struct {
	docs   [][]byte // list of documents to work on
	offset int64    // offset to start with
}

// RunWithWorkers start processing the input, uses multiple workers.
func (p LineProcessor) RunWithWorkers() error {

	var processingErr error

	// Setup communication channels.
	work := make(chan workPackage)
	updates := make(chan []Entry)
	done := make(chan bool)

	// collector runs the EntryWriter on all incoming batches.
	collector := func(ch chan []Entry, done chan bool) {
		for batch := range ch {
			if err := p.w(batch); err != nil {
				if p.Verbose {
					log.Printf("could not write batch: %v", err)
				}
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
					if p.Verbose {
						log.Printf("worker error: %v", err)
					}
					if p.IgnoreMissingKeys {
						if p.Verbose {
							log.Printf("ignoring missing key at offset: %d", offset)
							continue
						}
					}
					processingErr = err
					break
				}
				length := int64(len(b))
				entries = append(entries, Entry{key, offset, length})
				offset += length
			}
			updates <- entries
			if processingErr != nil {
				if p.Verbose {
					log.Printf("worker failed: %v", processingErr)
				}
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
	var offset = p.InitialOffset
	var blen int64
	batch := [][]byte{}

	var filesize int64
	var bar *progressbar.ProgressBar

	if f, ok := p.r.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return err
		}
		filesize = fi.Size()
		bar = progressbar.New(int(filesize))
	}

	for {
		b, err := br.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(b)) == 0 {
			continue
		}
		if len(batch) == p.BatchSize {
			if processingErr != nil {
				if p.Verbose {
					log.Printf("stopping early due to processing err: %v", processingErr)
				}
				// XXX: leaks resources.
				return processingErr
			}
			bb := make([][]byte, len(batch))
			copy(bb, batch)
			work <- workPackage{docs: bb, offset: offset}
			if _, ok := p.r.(*os.File); ok {
				bar.Add(int(offset))
			}
			offset += blen
			blen, batch = 0, nil
		}
		batch = append(batch, b)
		blen += int64(len(b))
	}

	bb := make([][]byte, len(batch))
	copy(bb, batch)
	work <- workPackage{docs: bb, offset: offset}

	if _, ok := p.r.(*os.File); ok {
		bar.Add(int(filesize))
		fmt.Println()
	}

	close(work)
	wg.Wait()
	close(updates)
	<-done

	return processingErr
}

// RegexpExtractor extract a key via regular expression.
type RegexpExtractor struct {
	Pattern *regexp.Regexp
}

// ExtractKey returns the key found in a byte slice. Never fails, just might
// return unexpected values.
func (e RegexpExtractor) ExtractKey(b []byte) (string, error) {
	return string(e.Pattern.Find(b)), nil
}

// ParsingExtractor actually parses the JSON and extracts a top-level key at the
// given path. This is slower than for example regular expressions, but not too much.
type ParsingExtractor struct {
	Key string
}

// ExtractKey extracts the key. Fails, if key cannot be found in the document.
func (e ParsingExtractor) ExtractKey(b []byte) (s string, err error) {
	dst := make(map[string]interface{})
	if err = json.Unmarshal(b, &dst); err != nil {
		return
	}
	if _, ok := dst[e.Key]; !ok {
		return "", fmt.Errorf("key %s not found in: %s", e.Key, string(bytes.TrimSpace(b)))
	}
	return renderString(dst[e.Key])
}

// ToplevelKeyExtractor parses a JSON object, where the actual object is nested
// under a top level key, e.g. {"mykey1": {"name": "alice"}}.
type ToplevelKeyExtractor struct{}

func (e ToplevelKeyExtractor) ExtractKey(b []byte) (s string, err error) {
	dst := make(map[string]interface{})
	if err = json.Unmarshal(b, &dst); err != nil {
		return
	}
	for k := range dst {
		return k, nil
	}
	return "", fmt.Errorf("no top level key: %v", string(b))
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
