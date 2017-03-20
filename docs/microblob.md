MICROBLOB 1 "MARCH 2017" "Leipzig University Library" "Manuals"
===============================================================

NAME
----

microblob - a simple key value server

SYNOPSIS
--------

`microblob` `-db` *dbpath* `-file` *blobfile* `-key` *string* [-append *file*] [-batch *NUM*]

`microblob` `-db` *dbpath* `-file` *blobfile* `-r` *pattern* [-append *file*] [-batch *NUM*]

`microblob` `-db` *dbpath* `-file` *blobfile* `-serve` [-log *file*] [-addr *hostport*]


DESCRIPTION
-----------

microblob serves documents from a file over HTTP. It finds and keeps the offsets
and lengths of the documents in a small embedded key-value store. When a key is
looked up, it will lookup the offset and length in the embedded key-value store
and then read the region directly from the file.

Performance will be dependent on how much of the original file can be kept in
the operating systems' page cache.

You can move data into the buffer cache with a simple cat(1) to null(4):

  `cat` *blobfile* `> /dev/null`

microblob can be updated via HTTP while running. Concurrent updates are not
supported, but won't cause errors, just block. After a successful update, the
new documents are appended to the *blobfile*. The store is append-only.

OPTIONS
-------

`-addr` *HOSTPORT*
  Hostport to listen (default "127.0.0.1:8820").

`-append` *FILE*
  Append this file to existing file and index into existing database.

`-backend` *NAME*
  Backend to use: leveldb, debug (default "leveldb").

`-batch`
  Number of lines in a batch (default 100000).

`-db` *FILE*
  Path to use for backend (default "data.db").

`-file` *FILE*
  File to index or serve.

`-key` *STRING*
  Key to extract, JSON, top-level only.

`-log` *FILE*
  Access log file, don't log if empty.

`-r` *PATTERN*
  Regular expression to use as key extractor.

`-serve`
  Serve file.

`-version`
  Show version and exit.

EXAMPLES
--------

Index a JSON file names example.ldj, use "id" field as key, then serve on port
12345 on localhost:

    $ microblob -db example.db -file example.ldj -key id
    $ microblob -db example.db -file example.ldj -serve -addr localhost:12345

Start an *empty* server, then index two documents with different keys, then
query. Neither `hello.db` not `hello.ldj` exist an the beginning:

    $ microblob -db hello.db -file hello.ldj -serve
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
