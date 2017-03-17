package microblob

import (
	"io"
	"log"
	"os"
	"sync"
)

// mu protects updates.
var mu sync.Mutex

// Append add a file to an existing blob file and adds their keys to the store. Not thread safe.
func Append(blobfn, fn string, backend Backend, kf KeyFunc) error {
	return AppendBatchSize(blobfn, fn, backend, kf, 100000)
}

// AppendBatchSize uses a given batch size.
func AppendBatchSize(blobfn, fn string, backend Backend, kf KeyFunc, size int) (err error) {
	mu.Lock()
	defer mu.Unlock()
	file, err := os.OpenFile(blobfn, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var offset int64

	if fn != "" {
		offset, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}

		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(file, f); err != nil {
			return err
		}
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return err
		}
	}

	processor := NewLineProcessor(file, backend.WriteEntries, kf)
	processor.BatchSize = size
	processor.InitialOffset = offset

	return processor.RunWithWorkers()
}
