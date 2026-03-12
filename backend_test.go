package microblob

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsAllZero(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want bool
	}{
		{"nil", nil, true},
		{"empty", []byte{}, true},
		{"all zeros", []byte{0, 0, 0}, true},
		{"has nonzero", []byte{0, 1, 0}, false},
		{"single zero", []byte{0}, true},
		{"single nonzero", []byte{42}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllZero(tt.in); got != tt.want {
				t.Errorf("IsAllZero(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestDebugBackend(t *testing.T) {
	var buf bytes.Buffer
	b := DebugBackend{Writer: &buf}

	entries := []Entry{
		{Key: "k1", Offset: 0, Length: 10},
		{Key: "k2", Offset: 10, Length: 20},
	}
	if err := b.WriteEntries(entries); err != nil {
		t.Fatal(err)
	}
	want := "k1\t0\t10\nk2\t10\t20\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}

	data, err := b.Get("anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("Get should return empty slice, got %v", data)
	}

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLevelDBBackendRoundTrip(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "test.blob")

	// Create a blob file with two JSON lines.
	line1 := []byte(`{"id":"abc","name":"Alice"}` + "\n")
	line2 := []byte(`{"id":"def","name":"Bob"}` + "\n")
	blob := append(line1, line2...)
	if err := os.WriteFile(blobPath, blob, 0644); err != nil {
		t.Fatal(err)
	}

	backend := &LevelDBBackend{
		Filename: filepath.Join(dir, "test.db"),
		Blobfile: blobPath,
	}
	defer backend.Close()

	entries := []Entry{
		{Key: "abc", Offset: 0, Length: int64(len(line1))},
		{Key: "def", Offset: int64(len(line1)), Length: int64(len(line2))},
	}
	if err := backend.WriteEntries(entries); err != nil {
		t.Fatal(err)
	}

	// Get existing key.
	data, err := backend.Get("abc")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(line1) {
		t.Errorf("Get(abc) = %q, want %q", data, line1)
	}

	// Get second key.
	data, err = backend.Get("def")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(line2) {
		t.Errorf("Get(def) = %q, want %q", data, line2)
	}

	// Get missing key.
	_, err = backend.Get("missing")
	if err == nil {
		t.Error("Get(missing) should return error")
	}

	// Count.
	count, err := backend.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestLevelDBBackendCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "test.blob")
	if err := os.WriteFile(blobPath, []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	backend := &LevelDBBackend{
		Filename: filepath.Join(dir, "test.db"),
		Blobfile: blobPath,
	}

	// Write something to open the DB, then get to open the blob.
	if err := backend.WriteEntries([]Entry{{Key: "x", Offset: 0, Length: 5}}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Get("x"); err != nil {
		t.Fatal(err)
	}

	// Close twice should not error.
	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
}
