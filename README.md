# cache
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/cache.svg)](https://pkg.go.dev/github.com/ammario/cache)

Package `cache` implements a generic LRU (least recently used) cache for Go.

```
go get github.com/ammario/cache
```

## Basic Usage
```go
c := NewLRU[string](cache.ConstantCost[string], 10)
c.Set("dog", "friendly")

// "friendly", "ok"
v, ok := c.Get("dog")
```
## Minimal disclaimer
This package is intentionally minimal (0 external dependencies, 5 exported symbols) and is not
designed to support every use case.