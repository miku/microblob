// Package microblob implements a thin layer above LevelDB to implement a key-value store.
package microblob

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

// ErrInvalidValue if a value is corrupted.
var ErrInvalidValue = errors.New("invalid entry")

// Entry associates a string key with a section in a file specified by offset and length.
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

// DebugBackend just writes the key, value and offsets to a given writer.
type DebugBackend struct {
	Writer io.Writer
}

// WriteEntries write entries as TSV to the given writer.
func (b DebugBackend) WriteEntries(entries []Entry) error {
	for _, e := range entries {
		if _, err := io.WriteString(b.Writer,
			fmt.Sprintf("%s\t%d\t%d\n", e.Key, e.Offset, e.Length)); err != nil {
			return err
		}

	}
	return nil
}

// Close is a noop.
func (b DebugBackend) Close() error { return nil }

// Get is a noop, always return nothing.
func (b DebugBackend) Get(key string) ([]byte, error) { return []byte{}, nil }

// LevelDBBackend writes entries into LevelDB.
type LevelDBBackend struct {
	Blobfile string
	blob     *os.File
	Filename string
	db       *leveldb.DB
	sync.Mutex
}

// Get retrieves the data for a given key.
func (b *LevelDBBackend) Get(key string) (data []byte, err error) {
	if err = b.openDatabase(); err != nil {
		return nil, err
	}

	var value []byte
	var offset, length int64

	if value, err = b.db.Get([]byte(key), nil); err != nil {
		return nil, err
	}
	if len(value) < 16 {
		return nil, ErrInvalidValue
	}
	if offset, err = binary.ReadVarint(bytes.NewBuffer(value[:8])); err != nil {
		return nil, err
	}
	if length, err = binary.ReadVarint(bytes.NewBuffer(value[8:])); err != nil {
		return nil, err
	}

	if err = b.openBlob(); err != nil {
		return nil, err
	}

	data = make([]byte, length)

	b.Lock()
	defer b.Unlock()

	if _, err = b.blob.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err = b.blob.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}

// Close closes database handle and blob file.
func (b *LevelDBBackend) Close() error {
	if b.db != nil {
		if err := b.db.Close(); err != nil {
			return err
		}
		b.db = nil
	}
	if b.blob != nil {
		if err := b.blob.Close(); err != nil {
			return err
		}
		b.blob = nil
	}
	return nil
}

// WriteEntries writes entries as batch into LevelDB. The value is fixed 16 byte
// slice, first 8 bytes represents the offset, last 8 bytes the length.
// https://play.golang.org/p/xwX8BmWtVl
func (b *LevelDBBackend) WriteEntries(entries []Entry) error {
	if err := b.openDatabase(); err != nil {
		return err
	}
	batch := new(leveldb.Batch)
	for _, entry := range entries {
		value := make([]byte, 16)
		binary.PutVarint(value[:8], entry.Offset)
		binary.PutVarint(value[8:], entry.Length)
		batch.Put([]byte(entry.Key), value)
	}
	return b.db.Write(batch, nil)
}

// openBlob opens the raw file. Save to call many times.
func (b *LevelDBBackend) openBlob() error {
	// TODO(miku): Store a SHA of the origin file in the blob store, compare with the
	// SHA of the currently used blob file, so we can warn the user if database and
	// file won't match.
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

// openDatabase creates a LevelDB handle. Save to call many times.
func (b *LevelDBBackend) openDatabase() error {
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
