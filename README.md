# microblob

microblob is a simplistic key-value store, that serves documents from a file
over HTTP. It is implemented in 728 lines of code and does not contain many
features.

[![DOI](https://zenodo.org/badge/82800367.svg)](https://zenodo.org/badge/latestdoi/82800367) [![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#active)

This project has been developed for [Project finc](https://finc.info) at [Leipzig University Library](https://ub.uni-leipzig.de).

```shell
$ cat file.ldj
{"id": "some-id-1", "name": "alice"}
{"id": "some-id-2", "name": "bob"}

$ microblob -key id file.ldj
INFO[0000] creating db fixtures/file.ldj.832a9151.db ...
INFO[0000] listening at http://127.0.0.1:8820 (fixtures/file.ldj.832a9151.db)
```

It supports fast rebuilds from scratch, as the preferred way to deploy this is
for a *build-once* *update-never* use case. It scales up and down with memory
and can serve hundred million documents and more.

Inspiration: [So what's wrong with 1975
programming?](http://varnish-cache.org/docs/trunk/phk/notes.html#so-what-s-wrong-with-1975-programming)
Idea: Instead of implementing complicated caching mechanisms, we hand over
caching completely to the operating system and try to stay out of its way.

Inserts are fast, since no data is actually moved. 150 million (1kB) documents
can be serveable within an hour.

* ㊗️ 2017-06-30 first 100 million requests served in production

# Update via curl

To send compressed data with curl:

```shell
$ curl -v --data-binary @- localhost:8820/update?key=id < <(gunzip -c fixtures/fake.ldj.gz)
...
```

# Usage

```shell
Usage of microblob:
  -addr string
          address to serve (default "127.0.0.1:8820")
  -backend string
          backend to use: leveldb, debug (default "leveldb")
  -batch int
          number of lines in a batch (default 100000)
  -key string
          key to extract, json, top-level only
  -log string
          access log file, don't log if empty
  -r string
          regular expression to use as key extractor
  -t    top level key extractor
  -version
          show version and exit

```

# What it doesn't do

* no deletions (microblob is currently append-only and does not care about
  garbage, so if you add more and more things, you will run out of space)
* no compression (yet)
* no security (anyone can query or update via HTTP)

# Installation

Debian and RPM packages: see [releases](https://github.com/miku/microblob/releases).

Or:

```shell
$ go get github.com/miku/microblob/cmd/...
```
