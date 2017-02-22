package main

import (
	"log"
	"os"

	"github.com/miku/microblob"
)

func loggingSink(ch <-chan []microblob.Entry, done chan bool) {
	for b := range ch {
		for _, entry := range b {
			log.Println(entry)
		}
	}
	done <- true
}

func main() {
	opts := microblob.ExtractorOptions{Key: "id", BatchSize: 1000}
	if err := microblob.Extract(os.Stdin, microblob.NewlineDelimitedJSON, loggingSink, opts); err != nil {
		log.Fatal(err)
	}
}
