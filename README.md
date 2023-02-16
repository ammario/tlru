# lru-cache
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/lru-cache.svg)](https://pkg.go.dev/github.com/ammario/lru-cache)

Package `lru` implements a generic LRU (least recently used) cache for Go. The
cache is safe for concurrent use.

```
go get github.com/ammario/lru-cache
```

## LRU
```go
c := lru.New(lru.ConstantCost[string], 10)
c.Set("dog", 10)

// 10, true
v, ok := c.Get()
```

## Small scope
This package is intentionally minimal (0 external dependencies, 5 exported symbols) and is not
designed to support every use case.