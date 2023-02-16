# cache
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/cache.svg)](https://pkg.go.dev/github.com/ammario/cache)

Package `cache` implements a generic LRU (least recently used) cache for Go.

```
go get github.com/ammario/cache
```

## LRU
```go
c := lru.New(cache.ConstantCost[string], 10)
c.Set("dog", 10)

// 10, true
v, ok := c.Get()
```

## Minimal disclaimer
This package is intentionally minimal (0 external dependencies, 5 exported symbols) and is not
designed to support every use case.