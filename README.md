microblob
=========

Serve documents of a newline delimited (JSON) file via HTTP. Do not store the
contents, just the offsets and lengths of the documents. The input documents
must be newline delimited.

**Status**: Currently in beta testing, API and flags might change. Use at your own risk.

Sketch
------

```
                    Index a file into a database. Specify JSON key or regular expression (faster).
                               +             +                  +           +
                               |             |        +---------+    +------+
                               v             v        v              v

           $ microblob -file blobfile -db data.db [ -key record_id, -p 'ai-[\d]+-[-a-zA-Z0-9_]+' ]

           +---------------------------------------------------------------------------------+
           |                                                                                 |
           |                                                                                 |
+------->  | HTTP request  +-------> lookup offset and length +---------------->  LevelDB <-------+
           |                                                                                 |    |
           |                                              +   <----------------+             |    |
           |                                              |                                  |    |
           |                                              |                                  |    |
<-------+  | HTTP response <-------+ seek and read <------+                                  |    |
           |                                                                                 |    |
           |                           ^  +                                                  |    |
           |                           |  |                                  --microblob     |    |
           +---------------------------------------------------------------------------------+    |
                                       |  |                                                       |
                                       |  |          +--------------------------------------------+
                                       |  |          |
                                       |  |          |
                                       |  v          |
                                                     +
                    $ microblob -file blobfile -db data.db -serve -addr 0.0.0.0:8820

                                                              ^
                                     +------------------------+
                                     |
                                  Serve a file on a specific address.

```

The goal is to serve a large number of keys, while being memory efficient and
fast to index. Creating a blob database with 120 million entries takes about an
hour, consumes little memory during preprocessing and only a few GB disk space
and will be served fast from memory, as soon as the OS
[caches](http://www.makelinux.net/books/lkd2/ch15)
parts of the blob file.

It should be possible to use this setup as is for slightly larger settings as well.

Inspiration: [So what's wrong with 1975 programming?](http://varnish-cache.org/docs/trunk/phk/notes.html#so-what-s-wrong-with-1975-programming)

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
        access log file, stderr if empty
  -r string
        regular expression to use as key extractor
  -serve
        serve file
  -version
        show version and exit

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

Example setup
-------------

![](https://raw.githubusercontent.com/miku/microblob/master/docs/asciicast.gif)

Performance
-----------

```shell
$ ll -h fixtures/example.ldj
-rw-rw-r-- 1 zzz zzz 120G Feb 22 15:35 fixtures/example.ldj

$ wc -l fixtures/example.ldj
118627938 fixtures/example.ldj

$ time microblob -db data.db -file fixtures/example.ldj -key finc.record_id
...
real    68m26.039s
user    58m47.116s
sys      3m21.976s

$ microblob -db data.db -file fixtures/example.ldj -serve
...
```

Ad-hoc benchmarks with [ab](https://httpd.apache.org/docs/2.4/programs/ab.html) and [hey](https://github.com/rakyll/hey).

```shell
$ ab -c 10 -n 10000 http://127.0.0.1:8820/ai-121-b2FpOmFyWGl2Lm9yZzowNzA0LjAwNTA
This is ApacheBench, Version 2.3 <$Revision: 1706008 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient)
Completed 1000 requests
Completed 2000 requests
Completed 3000 requests
Completed 4000 requests
Completed 5000 requests
Completed 6000 requests
Completed 7000 requests
Completed 8000 requests
Completed 9000 requests
Completed 10000 requests
Finished 10000 requests


Server Software:
Server Hostname:        127.0.0.1
Server Port:            8820

Document Path:          /ai-121-b2FpOmFyWGl2Lm9yZzowNzA0LjAwNTA
Document Length:        1576 bytes

Concurrency Level:      10
Time taken for tests:   0.445 seconds
Complete requests:      10000
Failed requests:        0
Total transferred:      16950000 bytes
HTML transferred:       15760000 bytes
Requests per second:    22480.30 [#/sec] (mean)
Time per request:       0.445 [ms] (mean)
Time per request:       0.044 [ms] (mean, across all concurrent requests)
Transfer rate:          37211.04 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       0
Processing:     0    0   0.1      0       2
Waiting:        0    0   0.1      0       2
Total:          0    0   0.1      0       3

Percentage of the requests served within a certain time (ms)
  50%      0
  66%      0
  75%      0
  80%      0
  90%      1
  95%      1
  98%      1
  99%      1
 100%      3 (longest request)
```

[Hey](https://github.com/rakyll/hey)!

```shell
$ hey -n 10000 http://localhost:8820/ai-48-R0xJUF9fTmpneU9UTTFPVUJBUURZNE1qa3pOVGs
All requests done.

Summary:
  Total:	0.2991 secs
  Slowest:	0.0326 secs
  Fastest:	0.0001 secs
  Average:	0.0014 secs
  Requests/sec:	33433.4975

Status code distribution:
  [200]	10000 responses

Response time histogram:
  0.000 [1]	|
  0.003 [9369]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.007 [519]	|∎∎
  0.010 [75]	|
  0.013 [25]	|
  0.016 [7]	|
  0.020 [2]	|
  0.023 [0]	|
  0.026 [1]	|
  0.029 [0]	|
  0.033 [1]	|

Latency distribution:
  10% in 0.0003 secs
  25% in 0.0006 secs
  50% in 0.0011 secs
  75% in 0.0017 secs
  90% in 0.0028 secs
  95% in 0.0036 secs
  99% in 0.0068 secs
```

Debug backend
-------------

```shell
$ ./microblob -backend debug -file fixtures/fake.ldj -key "id" | head -10
id-0	0	32
id-1	32	32
id-2	64	32
id-3	96	32
id-4	128	32
id-5	160	32
id-6	192	32
id-7	224	32
id-8	256	32
id-9	288	32
```

TODO
----

- [x] stats route (middleware)
- [x] logging (middleware)
- [x] simple appends
- [x] other possible backends: sqlite3, bdb, bolt; bolt hung at 113M records; debug later

Possible append usage:

```shell
$ microblob -db data.db -file example.ldj -append extra.ldj -key "id"
```

Would append file to a existing file and add keys to existing db. Not possible
to have multiple processes with leveldb. Take down, append, serve.
