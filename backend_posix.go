// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package microblob

import (
	"bytes"
	"encoding/binary"
	"syscall"
)

// Get retrieves the data for a given key, using pread(2).
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

	_, err = syscall.Pread(int(b.blob.Fd()), data, offset)

	return data, err
}
