package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miku/microblob"
	"github.com/thoas/stats"
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
		metrics := stats.New()
		blobHandler := metrics.Handler(
			microblob.WithLastResponseTime(
				&microblob.BlobHandler{Backend: backend}))

		r := mux.NewRouter()
		r.Handle("/debug/vars", http.DefaultServeMux)
		r.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(metrics.Data()); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "microblob",
				"version": microblob.Version,
				"stats":   fmt.Sprintf("http://%s/stats", r.Host),
				"vars":    fmt.Sprintf("http://%s/debug/vars", r.Host),
			}); err != nil {
				http.Error(w, "could not serialize", http.StatusInternalServerError)
			}
		})
		r.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
			if c, ok := backend.(microblob.Counter); ok {
				if count, err := c.Count(); err == nil {
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"count": count,
					}); err != nil {
						http.Error(w, "could not serialize", http.StatusInternalServerError)
					}
				}
			} else {
				http.Error(w, "not implemented", http.StatusNotFound)
			}
		})
		r.Handle("/update", microblob.UpdateHandler{Backend: backend, Blobfile: *blobfile})
		r.Handle("/blob", blobHandler)     // Legacy route.
		r.Handle("/{key:.+}", blobHandler) // Preferred.

		loggedRouter := handlers.LoggingHandler(loggingWriter, r)

		log.Printf("serving blobs from %[1]s on %[2]s, metrics at %[2]s/stats and %[2]s/debug/vars", *blobfile, *addr)
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
