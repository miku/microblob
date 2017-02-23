microblob
=========

Serve JSON from file via HTTP. Do not store the blobs in a key-value store
again, just the offset and lengths of the documents inside a file.

```
           +-------------------------------------------------------------------------------+
           |                                                                               |
           |                                                                               |
+------->  | HTTP request  +-------> lookup offset and length +---------------->  LevelDB  |
           |                                                                               |
           |                                              +   <----------------+           |
           |                                              |                                |
           |                                              |                                |
<-------+  | HTTP response <-------+ seek and read <------+                                |
           |                                                                               |
           |                           ^  +                                                |
           |                           |  |                                                |
           |                           |  |                                                |
           |                           |  |                                                |
           |                           |  |                                                |
           +-------------------------------------------------------------------------------+
                                       |  |
                                       |  v

                                     blobfile

```

Usage
-----

```shell
$ microblob -h
Usage of microblob:
  -addr string
          address to serve (default "127.0.0.1:8820")
  -backend string
          backend to use, currently only leveldb (default "leveldb")
  -batch int
          number of lines in a batch (default 100000)
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
