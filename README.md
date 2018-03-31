[![Build Status](https://travis-ci.org/valyala/gozstd.svg)](https://travis-ci.org/valyala/gozstd)
[![GoDoc](https://godoc.org/github.com/valyala/gozstd?status.svg)](http://godoc.org/github.com/valyala/gozstd)
[![Go Report](https://goreportcard.com/badge/github.com/valyala/gozstd)](https://goreportcard.com/report/github.com/valyala/gozstd)

# gozstd - go wrapper for [zstd](http://facebook.github.io/zstd/)

Features:

  * [Simple API](https://godoc.org/github.com/valyala/gozstd).
  * Optimized for speed. The API may be easily used in zero allocations mode.
  * Optimized for high concurrency.
  * Proper [Writer.Flush](https://godoc.org/github.com/valyala/gozstd#Writer.Flush)
    for network apps.
