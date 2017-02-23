microblob
=========

Serve JSON from file via HTTP. Do not store the blobs in a key-value store
again, just the offset and lengths of the documents inside a file.

```
request key -> find offset and length -
                 for Key in backend     \
<- response                               \
       ^                             seek and read data
       |                                from file
       |                                  /
       ----------------------------------´

```

Usage
-----

```shell
$ microblob -h
Usage of microblob:
  -addr string
        address to serve (default "127.0.0.1:8820")
  -backend string
        backend to use: tsv, leveldb, sqlite (default "leveldb")
  -db string
        filename to use for backend (default "data.db")
  -file string
        file to index or serve
  -key string
        key to extract
  -r string
        regular expression to use as key extractor
  -serve
        serve file
```

```shell
$ microblob -db data.db -file fixtures/1000.ldj -key finc.record_id
$ microblob -db data.db -file fixtures/1000.ldj -serve
$ curl -s localhost:8820/ai-121-b2FpOmFyWGl2Lm9yZzowNzA0LjAwMjQ | jq .
{
  "finc.format": "ElectronicArticle",
  "finc.mega_collection": "Arxiv",
  "finc.record_id": "ai-121-b2FpOmFyWGl2Lm9yZzowNzA0LjAwMjQ",
  "finc.source_id": "121",
  "rft.atitle": "Formation of quasi-solitons in transverse confined ferromagnetic film   media",
  "rft.jtitle": "Arxiv",
  ...
  "url": [
    "http://arxiv.org/abs/0704.0024"
  ],
  "x.subjects": [
    "Nonlinear Sciences - Pattern Formation and Solitons"
  ]
}
```

Performance
-----------

```shell
$ ll -h fixtures/example.ldj
-rw-rw-r-- 1 zzz zzz 120G Feb 22 15:35 fixtures/example.ldj

$ wc -l fixtures/example.ldj
118627938 fixtures/example.ldj

$ time microblob -db data.db -file fixtures/example.ldj -key finc.record_id

```
