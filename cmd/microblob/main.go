package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/miku/microblob"

	_ "expvar"
)

// Version of application.
const Version = "0.1.7"

func main() {
	pattern := flag.String("r", "", "regular expression to use as key extractor")
	keypath := flag.String("key", "", "key to extract, json, top-level only")
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

	var backend microblob.Backend

	switch *dbname {
	case "debug":
		backend = microblob.DebugBackend{Writer: os.Stdout}
	default:
		backend = &microblob.LevelDBBackend{
			Filename: *dbfile,
			Blobfile: *blobfile,
		}
	}

	defer func() {
		if err := backend.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if *serve {
		handler := microblob.WithStats(&microblob.BlobHandler{Backend: backend})
		http.Handle("/", handler)
		log.Printf("serving blobs from %s on %s", *blobfile, *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			log.Fatal(err)
		}
	}

	var extractor microblob.KeyExtractor

	switch {
	case *pattern != "":
		extractor = microblob.RegexpExtractor{
			Pattern: regexp.MustCompile(*pattern),
		}
	case *keypath != "":
		extractor = microblob.ParsingExtractor{
			Key: *keypath,
		}
	default:
		log.Fatal("key or pattern required")
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
