microblob
=========

Microblob serves JSON from file via HTTP.

----

package microblob

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// workerErr is set, when a worker fails. Winds down processing.
var workerErr error

// EntryProcessor is a function, that receives a batch of entries to process on
// and a channel to signal a successful end of processing.
type EntryProcessor func(ch <-chan []Entry, done chan bool)

// Worker is a function that processes the input. It receives a workPackage, which
// contains a sink to send the results to.
type Worker func(queue chan workPackage, wg *sync.WaitGroup)

// workPackage is a unit of work handed to a worker.
type workPackage struct {
	key    string       // key to extract
	docs   [][]byte     // documents to work on
	offset int64        // offset of the file from which the docs originate
	sink   chan []Entry // channel to send the results
}

// ExtractorOptions are configuration for the extraction.
type ExtractorOptions struct {
	Key       string // path to key to extract
	BatchSize int    // number of docs passed to a worker at once
}

// NewlineDelimitedJSON is a worker. It takes a workPackage, creates Entries from
// bytes and sends the result down the sink.
func NewlineDelimitedJSON(queue chan workPackage, wg *sync.WaitGroup) {
	defer wg.Done()
	for pkg := range queue {
		offset := pkg.offset
		var entries []Entry
		for _, b := range pkg.docs {
			dst := make(map[string]interface{})
			if err := json.Unmarshal(b, &dst); err != nil {
				workerErr = err
				break
			}
			if _, ok := dst[pkg.key]; !ok {
				workerErr = fmt.Errorf("key %s not found in: %s", pkg.key, string(b))
				break
			}
			s, err := renderString(dst[pkg.key])
			if err != nil {
				workerErr = err
				break
			}
			length := int64(len(b))
			entries = append(entries, Entry{s, offset, length})
			offset += length
		}
		pkg.sink <- entries
		if workerErr != nil {
			break
		}
	}
}

// Extract starts the extraction process in parallel. Worker and sink functions
// must be passed in. The worker function takes a uniq of work and passes the
// results to the sink. This way we can extract data from various formats, like
// newline-delimited JSON or some binary format. The given EntryProcessor controls
// the persistence options of may just log actions for debugging.
func Extract(r io.Reader, worker Worker, sink EntryProcessor, options ExtractorOptions) error {
	work := make(chan workPackage)
	updates := make(chan []Entry)
	done := make(chan bool)

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(work, &wg)
	}
	go sink(updates, done)

	br := bufio.NewReader(r)
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
		if len(bytes.TrimSpace(b)) == 0 {
			continue
		}
		if len(batch) == options.BatchSize {
			bb := make([][]byte, len(batch))
			copy(bb, batch)
			work <- workPackage{key: options.Key, docs: bb, offset: offset, sink: updates}
			offset += blen
			blen, batch = 0, nil
		}
		batch = append(batch, b)
		blen += int64(len(b))
	}

	bb := make([][]byte, len(batch))
	copy(bb, batch)
	work <- workPackage{key: options.Key, docs: bb, offset: offset, sink: updates}

	close(work)
	wg.Wait()
	close(updates)
	<-done

	return workerErr
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


----

func exampleProcessor(ch <-chan []microblob.Entry, done chan bool) {
	bw := bufio.NewWriter(os.Stdout)
	for batch := range ch {
		for _, entry := range batch {
			b, err := json.Marshal(entry)
			if err != nil {
				log.Fatal(err)
			}
			b = append(b, []byte("\n")...)
			if _, err := bw.Write(b); err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := bw.Flush(); err != nil {
		log.Fatal(err)
	}
	done <- true
}

func main() {
	// Setup options for processing.
	opts := microblob.ExtractorOptions{Key: "id", BatchSize: 10000}
	// Start the extraction process.
	if err := microblob.Extract(os.Stdin, microblob.NewlineDelimitedJSON, exampleProcessor, opts); err != nil {
		log.Fatal(err)
	}
}
