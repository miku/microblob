package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/handlers"
	"github.com/miku/microblob"
)

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
	logfile := flag.String("log", "", "access log file, don't log if empty")
	appendfile := flag.String("append", "", "append this file to existing file and index into existing database")

	flag.Parse()

	if *version {
		fmt.Println(microblob.Version)
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

	var loggingWriter = ioutil.Discard

	if *logfile != "" {
		file, err := os.OpenFile(*logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		loggingWriter = file
		defer file.Close()
	}

	if *serve {
		log.Printf("microblob at %v", *addr)

		r := microblob.NewHandler(backend, *blobfile, loggingWriter)
		loggedRouter := handlers.LoggingHandler(loggingWriter, r)

		if err := http.ListenAndServe(*addr, loggedRouter); err != nil {
			log.Fatal(err)
		}
	}

	var extractor microblob.KeyExtractor

	switch {
	case *pattern != "":
		p, err := regexp.Compile(*pattern)
		if err != nil {
			log.Fatal(err)
		}
		extractor = microblob.RegexpExtractor{Pattern: p}
	case *keypath != "":
		extractor = microblob.ParsingExtractor{Key: *keypath}
	default:
		log.Fatal("key or pattern required")
	}

	if err := microblob.AppendBatchSize(*blobfile, *appendfile, backend, extractor.ExtractKey, *batchsize); err != nil {
		log.Fatal(err)
	}
}
