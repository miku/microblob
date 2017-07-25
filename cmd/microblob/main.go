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

	"crypto/sha1"

	"github.com/gorilla/handlers"
	"github.com/miku/microblob"
)

func main() {
	pattern := flag.String("r", "", "regular expression to use as key extractor")
	keypath := flag.String("key", "", "key to extract, json, top-level only")
	dbname := flag.String("backend", "leveldb", "backend to use: leveldb, debug")
	addr := flag.String("addr", "127.0.0.1:8820", "address to serve")
	batchsize := flag.Int("batch", 400000, "number of lines in a batch")
	version := flag.Bool("version", false, "show version and exit")
	logfile := flag.String("log", "", "access log file, don't log if empty")

	flag.Parse()

	if *version {
		fmt.Println(microblob.Version)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		log.Fatal("file to index and serve required")
	}

	blobfile := flag.Arg(0)

	if blobfile == "" {
		log.Fatal("need a file to index or serve")
	}

	var dbfile string

	if dbfile == "" {
		h := sha1.New()
		if _, err := fmt.Fprintf(h, "%s:%s:%s", *dbname, *keypath, *pattern); err != nil {
			log.Fatal(err)
		}
		dbfile = fmt.Sprintf("%s.%x.microdb", blobfile, h.Sum(nil))
	}

	var backend microblob.Backend

	switch *dbname {
	case "debug":
		backend = microblob.DebugBackend{Writer: os.Stdout}
	default:
		backend = &microblob.LevelDBBackend{
			Filename: dbfile,
			Blobfile: blobfile,
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

	// If dbfile does not exists, create it now.
	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		log.Printf("building file map (%s)", dbfile)
		var extractor microblob.KeyExtractor

		switch {
		case *pattern != "":
			p, err := regexp.Compile(*pattern)
			if err != nil {
				log.Fatal(err)
			}
			extractor = microblob.RegexpExtractor{Pattern: p}
			if err := microblob.AppendBatchSize(blobfile, "", backend, extractor.ExtractKey, *batchsize); err != nil {
				log.Fatal(err)
			}
		case *keypath != "":
			extractor = microblob.ParsingExtractor{Key: *keypath}
			if err := microblob.AppendBatchSize(blobfile, "", backend, extractor.ExtractKey, *batchsize); err != nil {
				log.Fatal(err)
			}
		}

	}

	log.Printf("listening at http://%v (%s)", *addr, dbfile)
	r := microblob.NewHandler(backend, blobfile)
	loggedRouter := handlers.LoggingHandler(loggingWriter, r)
	if err := http.ListenAndServe(*addr, loggedRouter); err != nil {
		log.Fatal(err)
	}
}
