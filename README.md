microblob
=========

Microblob serves JSON from file via HTTP.

Usage
-----

```
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
