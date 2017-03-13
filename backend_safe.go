// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris

package microblob

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Get retrieves the data for a given key.
// Raw timings of the operations:
// Cold:
//     time.Now(): 89ns
//     b.openDatabase(): 2.692719ms
//     b.db.Get: 5.441917ms
//     binary.ReadVarint: 5.456746ms
//     make([]byte, length): 5.473913ms
//     b.blob.Seek: 5.479361ms
//     b.blob.Read: 5.487422ms
// Warm:
//     time.Now(): 57ns
//     b.openDatabase(): 86.018µs
//     b.db.Get: 139.769µs
//     binary.ReadVarint: 155.258µs
//     make([]byte, length): 210.089µs
//     b.blob.Seek: 218.031µs
//     b.blob.Read: 252.66µs
//
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
