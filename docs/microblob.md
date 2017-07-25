MICROBLOB 1 "MARCH 2017" "Leipzig University Library" "Manuals"
===============================================================

NAME
----

microblob - a simplistic key value server

SYNOPSIS
--------

`microblob` `-key` *string* [-batch *NUM*] *blobfile*

`microblob` `-r` *pattern* [-batch *NUM*] *blobfile*

`microblob` [-log *file*] [-addr *hostport*] *blobfile*


DESCRIPTION
-----------

microblob serves JSON documents from a single file over HTTP. It finds and
keeps the offsets and lengths of the documents in a small embedded key-value
store. When a key is requested, it will lookup the offset and length in the
key-value store, seek to the offset and read from the file.

Performance will depend on how much of the file can be kept in the operating
systems' page cache.

You can move data into the page cache with a simple cat(1) to null(4):

  `cat` *blobfile* `> /dev/null`

microblob can be updated via HTTP while running. Concurrent updates are not
supported: they do not cause errors, just block. After a successful update, the
new documents are appended to the *blobfile*. At the moment microblob is
append-only.

The use case for microblob is the create-once, update-never case. A newline
delimited JSON file with 120M documents and a filesize of 130G can be servable
in 40 minutes, which amounts to 50k documents/s or 55M/s sustained inserts.

OPTIONS
-------

`-addr` *HOSTPORT*
  Hostport to listen (default "127.0.0.1:8820").

`-backend` *NAME*
  Backend to use: leveldb, debug (default "leveldb").

`-batch`
  Number of lines in a batch (default 100000).

`-key` *STRING*
  Key to extract, JSON, top-level only.

`-log` *FILE*
  Access log file, don't log if empty.

`-r` *PATTERN*
  Regular expression to use as key extractor.

`-version`
  Show version and exit.

EXAMPLES
--------

First, index a JSON file named example.ldj and use "id" field as key, then serve on port
12345 on localhost:

    $ microblob -key id example.ldj
    $ microblob -key id -addr localhost:12345 example.ldj

Start an *empty* server, then index two documents with different keys, then
query. Neither `hello.db` not `hello.ldj` exist an the beginning:

    $ microblob hello.ldj
    ...

    $ curl -XPOST -d '{"id": 1, "name": "alice"}' localhost:8820/update?key=id
    $ curl -XPOST -d '{"x-id": 2, "name": "bob"}' localhost:8820/update?key=x-id

    $ curl -s localhost:8820/1
    {"id": 1, "name": "alice"}

    $ curl -s localhost:8820/2
    {"x-id": 2, "name": "bob"}

DIAGNOSTICS
-----------

Live usage statistics are exposed over HTTP:

    $ curl -s localhost:8820/stats | jq .
    {
      "pid": 14701,
      "uptime": "7m21.527249914s",
      "uptime_sec": 441.527249914,
      "time": "2017-03-20 15:51:02.720553958 +0100 CET",
      "unixtime": 1490021462,
      "status_code_count": {},
      "total_status_code_count": {
        "200": 4
      },
      "count": 0,
      "total_count": 4,
      "total_response_time": "300.243µs",
      "total_response_time_sec": 0.000300243,
      "average_response_time": "75.06µs",
      "average_response_time_sec": 7.506e-05
    }

The response time of the last key query is exposed over HTTP as well:

    $ curl -s localhost:8820/debug/vars | jq .lastResponseTime
    5.6743e-05

BUGS
----

Please report bugs to https://github.com/miku/microblob/issues.

AUTHORS
-------

Martin Czygan <martin.czygan@uni-leipzig.de>

SEE ALSO
--------

curl(1), cat(1), null(4), pread(2), mincore(2), free(1), memcachedb(1)
