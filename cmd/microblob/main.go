// Exectable for microblob, can read options from flags or an ini file:
//
// file = /var/data/data.jsonlines
// port = 8820
// host = 0.0.0.0
// batchsize = 30000
//
package main

import (
	"crypto/sha1"
	_ "expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"

	"github.com/gorilla/handlers"
	"github.com/miku/microblob"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var (
	configFile        = flag.String("c", "", "load options from a config (ini) file")
	configFileSection = flag.String("s", "main", "the config file section to use")
	pattern           = flag.String("r", "", "regular expression to use as key extractor")
	toplevel          = flag.Bool("t", false, "top level key extractor")
	keypath           = flag.String("key", "", "key to extract, json, top-level only")
	dbname            = flag.String("backend", "leveldb", "backend to use: leveldb, debug")
	addr              = flag.String("addr", "127.0.0.1:8820", "address to serve")
	batchsize         = flag.Int("batch", 50000, "number of lines in a batch")
	version           = flag.Bool("version", false, "show version and exit")
	logfile           = flag.String("log", "", "access log file, don't log if empty")
	ignoreMissingKeys = flag.Bool("ignore-missing-keys", false, "ignore record, that do not have a the specified key")
	dbOnly            = flag.Bool("create-db-only", false, "build the database only, then exit")
	dbFile            = flag.String("db", "", "the root directory, by default: 1000.ldj -> 1000.ldj.05028f38.db (based on flags)")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println(microblob.Version)
		os.Exit(0)
	}
	var blobfile string
	log.Printf(*configFile)
	if *configFile != "" {
		// Load config file and set flag values.
		cfg, err := ini.Load(*configFile)
		if err != nil {
			log.Fatalf("could not load config file %s: %v", *configFile, err)
		}
		section := cfg.Section(*configFileSection)
		blobfile = section.Key("file").String()
		*keypath = section.Key("key").String()
		*pattern = section.Key("pattern").String()
		*toplevel, err = section.Key("toplevel").Bool()
		*dbFile = section.Key("db").String()
		*addr = section.Key("addr").String()
		*logfile = section.Key("log").String()
		*batchsize, err = section.Key("batch").Int()
	}
	if *configFile == "" && flag.NArg() == 0 {
		log.Fatal("file to index (and serve) required")
	}
	if blobfile == "" {
		blobfile = flag.Arg(0)
	}
	if blobfile == "" {
		log.Fatal("need a file to index or serve")
	}
	if *keypath == "" && *pattern == "" && !*toplevel {
		log.Fatal("need path, pattern or -t to identify key")
	}
	if *dbFile == "" {
		h := sha1.New()
		if _, err := fmt.Fprintf(h, "%s:%s:%s", *dbname, *keypath, *pattern); err != nil {
			log.Fatal(err)
		}
		*dbFile = fmt.Sprintf("%s.%.4x.db", blobfile, h.Sum(nil))
	}

	var backend microblob.Backend
	switch *dbname {
	case "debug":
		backend = microblob.DebugBackend{Writer: os.Stdout}
	default:
		backend = &microblob.LevelDBBackend{
			Filename: *dbFile,
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
	if _, err := os.Stat(*dbFile); os.IsNotExist(err) {
		log.Printf("creating db %s ...", *dbFile)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				log.Printf("%v -- cleaning up: %s", sig, *dbFile)
				if err := os.RemoveAll(*dbFile); err != nil {
					log.Fatal(err)
				}
				os.Exit(0)
			}
		}()
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
		case *toplevel:
			extractor = microblob.ToplevelKeyExtractor{}
		default:
			log.Fatal("exactly one key extraction method required: -r, -key or -t")
		}
		if err := microblob.AppendBatchSize(blobfile, "", backend,
			extractor.ExtractKey, *batchsize, *ignoreMissingKeys); err != nil {
			os.RemoveAll(*dbFile)
			log.Fatal(err)
		}
		close(c)
		signal.Stop(c)
	}
	if *dbOnly {
		os.Exit(0)
	}
	log.Printf("listening at http://%v (%s)", *addr, *dbFile)
	var (
		r            = microblob.NewHandler(backend, blobfile)
		loggedRouter = handlers.LoggingHandler(loggingWriter, r)
	)
	if err := http.ListenAndServe(*addr, loggedRouter); err != nil {
		log.Fatal(err)
	}
}
