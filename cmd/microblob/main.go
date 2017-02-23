package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/miku/microblob"
)

// Version of application.
const Version = "0.1.5"

// debugBackend just writes the key, value and offsets to standard output.
type debugBackend struct{}

func (b debugBackend) WriteEntries(entries []microblob.Entry) error {
	for _, e := range entries {
		fmt.Printf("%s\t%d\t%d\n", e.Key, e.Offset, e.Length)
	}
	return nil
}

func (b debugBackend) Close() error                   { return nil }
func (b debugBackend) Get(key string) ([]byte, error) { return []byte{}, nil }

func main() {
	pattern := flag.String("r", "", "regular expression to use as key extractor")
	keypath := flag.String("key", "", "key to extract")
	dbname := flag.String("backend", "leveldb", "backend to use: leveldb, debug")
	dbfile := flag.String("db", "data.db", "filename to use for backend")
	blobfile := flag.String("file", "", "file to index or serve")
	serve := flag.Bool("serve", false, "serve file")
	addr := flag.String("addr", "127.0.0.1:8820", "address to serve")
	batchsize := flag.Int("batch", 100000, "number of lines in a batch")
	version := flag.Bool("version", false, "show version and exit")

	flag.Parse()

	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *blobfile == "" {
		log.Fatal("need a file to index or serve")
	}

	// Choose backend.
	var backend microblob.Backend

	switch *dbname {
	default:
		backend = &microblob.LevelDBBackend{Filename: *dbfile, Blobfile: *blobfile}
	case "debug":
		backend = debugBackend{}
	}
	defer func() {
		if err := backend.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Serve content.
	if *serve {
		http.Handle("/", &microblob.BlobHandler{Backend: backend})
		log.Printf("serving blobs from %s on %s", *blobfile, *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			log.Fatal(err)
		}
	}

	// Index content.
	if *pattern == "" && *keypath == "" {
		log.Fatal("key or pattern required")
	}

	var extractor microblob.KeyExtractor

	if *pattern != "" {
		extractor = microblob.RegexpExtractor{Pattern: regexp.MustCompile(*pattern)}
	}
	if *keypath != "" {
		extractor = microblob.ParsingExtractor{Key: *keypath}
	}

	file, err := os.Open(*blobfile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	processor := microblob.NewLineProcessorBatchSize(
		file,
		backend.WriteEntries,
		extractor.ExtractKey,
		*batchsize,
	)
	if err := processor.RunWithWorkers(); err != nil {
		log.Fatal(err)
	}
}
