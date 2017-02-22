package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

// Entry associates a key with a section in a file specified by offset and length.
type Entry struct {
	Key    string `json:"k"`
	Offset int64  `json:"o"`
	Length int64  `json:"l"`
}

// KeyFunc extracts a key from a blob.
type KeyFunc func([]byte) (string, error)

// EntryFunc turns a blob to an entry.
type EntryFunc func([]byte) (Entry, error)

// EntryWriter writes entries to some storage, e.g. a file or a database.
type EntryWriter func(entries []Entry) error

// LineProcessor read a line, extracts the key and writes entries.
type LineProcessor struct {
	r io.Reader   // input data
	f KeyFunc     // extracts a string key from a byte blob
	w EntryWriter // serializes entries
}

// Run start processing the input.
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

func main() {
	processor := LineProcessor{
		r: os.Stdin,
		f: func(b []byte) (string, error) {
			return fmt.Sprintf("x-%d", len(b)), nil
		},
		w: func(entries []Entry) error {
			for _, e := range entries {
				log.Println(e)
			}
			return nil
		}}
	if err := processor.Run(); err != nil {
		log.Fatal(err)
	}
}
