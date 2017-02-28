// Package microblob implements a thin layer above LevelDB to implement a key-value store.
package microblob

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

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

// LevelDBBackend writes entries into LevelDB.
type LevelDBBackend struct {
	Blobfile string
	blob     *os.File
	Filename string
	db       *leveldb.DB
	sync.Mutex
}

// Get retrieves the data for a given key.
func (b *LevelDBBackend) Get(key string) ([]byte, error) {
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

// Close closes database handle and blob file.
func (b *LevelDBBackend) Close() error {
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
