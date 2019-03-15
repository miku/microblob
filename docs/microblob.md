MICROBLOB 1 "MARCH 2017" "Leipzig University Library" "Manuals"
===============================================================

NAME
----

microblob - a simplistic key value server

SYNOPSIS
--------

`microblob` `-key` *string* [-addr *HOSTPORT*] [-batch *NUM*] [-log *file*] *blobfile*

`microblob` `-r` *pattern* [-addr *HOSTPORT*] [-batch *NUM*] [-log *file*] *blobfile*

`microblob` `-t` [-addr *HOSTPORT*] [-batch *NUM*] [-log *file*] *blobfile*

DESCRIPTION
-----------

microblob serves documents from a single file (of newline delimited JSON) over
HTTP. It finds and keeps the offsets and lengths of the documents in a small
embedded database. When a key is requested, it will lookup the offset and
length, seek to the offset and read from the file.

The use case for microblob is the *create-once*, *update-never* case. A file
with 120M documents (130GB) is servable in 40 minutes (50k docs/s or 55M/s
sustained *inserts*).

Serving Performance will depend on how much of the file can be kept in the
operating systems' page cache.

You can move data into the page cache with a simple cat(1) to null(4):

  `cat` *blobfile* `> /dev/null`

microblob can be updated via HTTP while running. Concurrent updates are not
supported: they do not cause errors, just block. After a successful update, the
new documents are appended to the *blobfile*. Currently microblob is
*append-only*.

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

`-t`
  Top level key extractor.

`-version`
  Show version and exit.

EXAMPLES
--------

Index and serve (on port localhost:12345) a JSON file named *example.ldj* and
use *id* field as key:

    $ microblob -key id -addr localhost:12345 example.ldj
    ...

Start with an *empty* blobfile, then index two documents with different keys,
then query (hello.ldj does not exists at the beginning):

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

Get current number of documents (might take a few seconds):

    $ curl -s localhost:8820/count
    {"count": 12391823}

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
    0.001238

BUGS
----

Please report bugs to <https://github.com/miku/microblob/issues>.

AUTHORS
-------

Martin Czygan <martin.czygan@uni-leipzig.de>

SEE ALSO
--------

curl(1), cat(1), null(4), pread(2), mincore(2), free(1), memcachedb(1)
