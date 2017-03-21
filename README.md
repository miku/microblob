microblob
=========

microblob is a key-value store that serves documents from a file over HTTP.

The main use case was to quickly serve a single file of newline delimited JSON
documents over HTTP. For example, if we want to serve documents by the property
attribute value *id*, we can say:

```shell
$ cat path/to/file.ldj
{"id": "some-id-1", "name": "alice"}
{"id": "some-id-2", "name": "bob"}
...

$ microblob -db test.db -file path/to/file.ldj -key id
$ microblob -db test.db -file path/to/file.ldj -serve
...
$ curl localhost:8820/some-id-1
{"id": "some-id-1", "name": "alice"}
```

It supports fast rebuilds from scratch and additional documents can be added
easily. It scales up and down with memory and can serve hundred million
documents and more.

Inspiration: [So what's wrong with 1975
programming?](http://varnish-cache.org/docs/trunk/phk/notes.html#so-what-s-wrong-with-1975-programming).
Idea: Instead of implementing complicated caching mechanisms, we hand over
caching completely to the operating system and try to stay out of its way.

Inserts are super fast, since no data is actually moved. A 120G file containing
about 100 million documents can be serveable within an hour.

More examples
-------------

```shell
$ microblob -db test.db -file test.file -serve
2017/03/20 11:19:36 serving blobs from test.file on 127.0.0.1:8820 ...

$ curl -s localhost:8820 | jq .
{
  "name": "microblob",
  "stats": "http://localhost:8820/stats",
  "vars": "http://localhost:8820/debug/vars",
  "version": "0.1.16"
}

$ curl -v -XPOST -d '{"id": 1, "name": "alice"}' "http://localhost:8820/update?key=id"
$ curl -s  "http://localhost:8820/1" | jq .
{
  "id": 1,
  "name": "alice"
}

$ cat fixtures/fake-00-09.ldj
{"name": "hello", "id": "id-0"}
{"name": "hello", "id": "id-1"}
{"name": "hello", "id": "id-2"}
{"name": "hello", "id": "id-3"}
{"name": "hello", "id": "id-4"}
{"name": "hello", "id": "id-5"}
{"name": "hello", "id": "id-6"}
{"name": "hello", "id": "id-7"}
{"name": "hello", "id": "id-8"}
{"name": "hello", "id": "id-9"}

$ curl -v -XPOST --data-binary '@fixtures/fake-00-09.ldj' "http://localhost:8820/update?key=id"
$ curl -s localhost:8820/id-5 | jq .
{
  "name": "hello",
  "id": "id-5"
}
```

One use case for microblob was to serve data from a single file without (or
only a few) updates. Given a JSON file to serve, this will be the fastest
method to index and serve the file:

```shell
$ cat fixtures/fake-00-09.ldj
{"name": "hello", "id": "id-0"}
{"name": "hello", "id": "id-1"}
{"name": "hello", "id": "id-2"}
{"name": "hello", "id": "id-3"}
{"name": "hello", "id": "id-4"}
{"name": "hello", "id": "id-5"}
{"name": "hello", "id": "id-6"}
{"name": "hello", "id": "id-7"}
{"name": "hello", "id": "id-8"}
{"name": "hello", "id": "id-9"}

$ microblob -db test.db -file fixtures/fake-00-09.ldj -key id
$ microblob -db test.db -file fixtures/fake-00-09.ldj -serve
2017/03/20 11:19:36 serving blobs from fixtures/fake-00-09.ldj on 127.0.0.1:8820 ...
```

Usage
-----

```shell
$ microblob -h
Usage of microblob:
  -addr string
        address to serve (default "127.0.0.1:8820")
  -append string
        append this file to existing file and index into existing database
  -backend string
        backend to use: leveldb, debug (default "leveldb")
  -batch int
        number of lines in a batch (default 100000)
  -db string
        filename to use for backend (default "data.db")
  -file string
        file to index or serve
  -key string
        key to extract, json, top-level only
  -log string
        access log file, don't log if empty
  -r string
        regular expression to use as key extractor
  -serve
        serve file
  -version
        show version and exit

```
