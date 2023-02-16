# lru-cache
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/lru-cache.svg)](https://pkg.go.dev/github.com/ammario/lru-cache)

Package `lru` implements a generic LRU (least recently used) cache for Go. The
cache is safe for concurrent use.

```
go get github.com/ammario/lru-cache
```

## LRU

Basic example:
```go
// This cache can store up to 100 values.
c := lru.New(lru.ConstantCost[int], 100)
c.Set("dog", 3.14)

// 3.14, true
v, ok := c.Get()
```

Dynamic costs:
```go
// This cache can store up to 100 bytes.
c := New(
    func(v string) int {
        return len(v)
    },
    100,
)
c.Set("some_key", "some value")
```

## Small scope
This package is intentionally minimal (0 external dependencies, 5 exported symbols) and is not
designed to support every use case.