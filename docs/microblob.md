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

There exist tools for inquiring the number of cached pages for a given file or
directory. One such tool is: https://github.com/tobert/pcstat

> A common question when tuning databases and other IO-intensive applications
> is, "is Linux caching my data or not?" pcstat gets that information for you
> using the mincore(2) syscall.

UPDATES
-------

microblob can be updated via HTTP while running. Concurrent updates are not
supported: they do not cause errors, just block. After a successful update, the
new documents are appended to the *blobfile*. Currently microblob is
*append-only*.

If you need frequent updates, consider something else, e.g.  Badger, RocksDB,
memcachedb, or one of the many others
https://db-engines.com/en/ranking/key-value+store.

OPTIONS
-------

`-addr` *HOSTPORT*
  Hostport to listen (default "127.0.0.1:8820").

`-backend` *NAME*
  Backend to use: leveldb, debug (default "leveldb").

`-batch`
  Number of lines in a batch (default 100000).

`-c string`
  Load options from a config (ini) file

`-create-db-only`
  Build the database only, then exit.

`-db string`
  The root directory, by default: 1000.ldj -> 1000.ldj.05028f38.db (based on flags).

`-key` *STRING*
  Key to extract, JSON, top-level only.

`-log` *FILE*
  Access log file, don't log if empty.

`-r` *PATTERN*
  Regular expression to use as key extractor.

`-s string`
  The config file section to use (default "main").

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

FILES
-----

Since 0.2.12 it is possible to put options into a configuration file. This
features was added to let microblob be managed by systemd.

On installation from package, a default config file is placed at
`/etc/microblob/microblob.ini` and systemd unit is provided. The config file
can contain multiple sections, with `main` being used by default. Except for
`file` entries are optional an will use default values.

```
[main]

file = /var/microblob/date-2020-08-10.ldj
db = /var/microblob/date-2020-08-10.ldj.28ed2061.db
addr = 172.18.113.99:8820
batch = 30000
key = finc.id
log = /var/log/microblob.log

```

BUGS
----

Please report bugs to <https://github.com/miku/microblob/issues>.

AUTHORS
-------

Martin Czygan <martin.czygan@uni-leipzig.de>

SEE ALSO
--------

curl(1), cat(1), null(4), pread(2), mincore(2), free(1), memcachedb(1)
