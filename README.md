# Tiered Cache for Go

[![GoDoc](https://godoc.org/github.com/spothero/tieredcache?status.svg)](https://godoc.org/github.com/spothero/tieredcache)
[![Build Status](https://travis-ci.org/spothero/tieredcache.png?branch=master)](https://travis-ci.org/spothero/tieredcache)
[![codecov](https://codecov.io/gh/spothero/tieredcache/branch/master/graph/badge.svg)](https://codecov.io/gh/spothero/tieredcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/spothero/tieredcache)](https://goreportcard.com/report/github.com/spothero/tieredcache)


tieredcache composes a local in-memory cache ([BigCache](https://github.com/allegro/bigcache)) with Clustered Redis ([redigo](https://github.com/gomodule/redigo)/[redisc](https://github.com/mna/redisc)). Local
cache is checked first. If a result is not found, the redis cluster is checked. Cache sets are
performed on both local and remote cache.

API documentation and examples can be found in the [GoDoc](https://godoc.org/github.com/spothero/tieredcache)

## License
Apache 2
