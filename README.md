# tlru
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/tlru.svg)](https://pkg.go.dev/github.com/ammario/tlru)

Package `tlru` implements TLRU (Time Aware Least Recently Used)
cache for Go.

Features:

* Uses generics for type-safety
* Memory-backed
* Safe for concurrent use
* No background threads

```
go get github.com/ammario/tlru@master
```

## Examples

Basic example:
```go
// This cache can store up to 100 values.
c := tlru.New[string](tlru.ConstantCost[int], 100)
c.Set("dog", 3.14, time.Second)

// 3.14, ~time.Now().Add(time.Second), true
v, deadline, ok := c.Get("dog")
```

Dynamic costs:
```go
// This cache can store up to 100 bytes.
c := New[string](
    func(v string) int {
        return len(v)
    },
    100,
)
c.Set("some_key", "some value", time.Minute)
```

## Eviction

Cache eviction occurs during:

- Calls to `Set()` 
- Calls to `Evict()`
- Calls to `Get()` (for that key only) 

Cache eviction is fast because the LRU and TTL indices are sorted. In most
cases, a call to the evictor only touches a few entries. Calling `Evict()`
directly is usually unnecessary.

## Benchmarks
```
goos: darwin
goarch: amd64
pkg: github.com/ammario/tlru
cpu: VirtualApple @ 2.50GHz
Benchmark_TLRU_Get-10    	 7117467	       141.9 ns/op
Benchmark_TLRU_Set-10    	 1222298	       990.9 ns/op
```