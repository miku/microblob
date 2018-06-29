package microblob

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// mu protects updates.
var mu sync.Mutex

// Append add a file to an existing blob file and adds their keys to the store.
func Append(blobfn, fn string, backend Backend, kf KeyFunc) error {
	return AppendBatchSize(blobfn, fn, backend, kf, 100000, false)
}

// AppendBatchSize uses a given batch size.
func AppendBatchSize(blobfn, fn string, backend Backend, kf KeyFunc, size int, ignoreMissingKeys bool) (err error) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(blobfn, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
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
	processor.Verbose = true
	processor.IgnoreMissingKeys = ignoreMissingKeys

	if err = processor.RunWithWorkers(); err != nil {
		if fn != "" {
			if terr := os.Truncate(blobfn, offset); terr != nil {
				return fmt.Errorf("processing and truncate failed: %v, %v", err, terr)
			}
		}
	}
	return err
}
