package microblob

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"time"
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
	r         io.Reader   // input data
	f         KeyFunc     // extracts a string key from a byte blob
	w         EntryWriter // serializes entries
	BatchSize int         // number of lines in a batch
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
