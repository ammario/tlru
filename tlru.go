package tlru

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/ammario/tlru/internal/doublelist"
	"github.com/armon/go-radix"
)

// Coster is a function that returns the approximate memory cost of a
// given value.
type Coster[T any] func(v T) int

// ConstantCost always returns 1.
func ConstantCost[T any](_ T) int {
	return 1
}

// dataWithKey bundles data with its reference key.
// This structure allows for reverse lookup from the doubly-linked list to the index.
type dataWithKey[K comparable, V any] struct {
	data     V
	key      K
	deadline time.Time
}

// Cache implements a time aware least-frequently-used cache structure.
// When the cache exceeds a given cost limit, the oldest chunks of data are discarded.
type Cache[K comparable, V any] struct {
	mu sync.Mutex

	index map[K]*doublelist.Node[dataWithKey[K, V]]
	// lruList contains entries in order of least-recently-used to most-recently-used.
	lruList *doublelist.List[dataWithKey[K, V]]
	// ttlTrie contains entries in order of expires first to expires last.
	// Entries are sorted by their UnixNano deadline.
	ttlTrie *radix.Tree
	// coster allows for user-defined relative weighting of cache members.
	coster Coster[V]
	cost   int
	// costLimit sets the maximum storage cost of the cache.
	costLimit int
}

// New instantiates a ready-to-use LRU cache. It is safe for concurrent use. If cost is nil,
// a constant cost of 1 is assumed.
// Use -1 for costLimit to disable cost limiting.
func New[K comparable, V any](cost Coster[V], costLimit int) *Cache[K, V] {
	if cost == nil {
		cost = ConstantCost[V]
	}
	return &Cache[K, V]{
		index:     make(map[K]*doublelist.Node[dataWithKey[K, V]]),
		lruList:   &doublelist.List[dataWithKey[K, V]]{},
		ttlTrie:   radix.New(),
		coster:    cost,
		costLimit: costLimit,
	}
}

// strconv is too expensive
func parseDeadlineKey(s string) time.Time {
	return time.Unix(0, int64(binary.BigEndian.Uint64([]byte(s))))
}

func formatDeadlineKey(t time.Time) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(t.UnixNano()))
	return string(b[:])
}

func (l *Cache[K, V]) delete(key K) int {
	node, ok := l.index[key]
	if !ok {
		return 0
	}
	l.lruList.Pop(node)
	costSaving := l.coster(node.Data.data)
	l.cost -= costSaving

	deadlineKey := formatDeadlineKey(node.Data.deadline)
	_, ok = l.ttlTrie.Delete(deadlineKey)
	if !ok {
		// Something is very, very wrong.
		panic(fmt.Sprintf("key %q not deleted? cache corrupt", deadlineKey))
	}
	delete(l.index, key)
	return costSaving
}

func (l *Cache[K, V]) evictExpires() int {
	var ds int
	now := time.Now()
	for {
		deadlineKey, v, ok := l.ttlTrie.Minimum()
		if !ok {
			return ds
		}

		expiresAt := parseDeadlineKey(deadlineKey)
		if expiresAt.After(now) {
			// Abort, we have reached valid keys.
			return ds
		}

		k := v.(K)
		ds += l.delete(k)
	}
}

func (l *Cache[K, V]) evictOverages() int {
	if l.costLimit < 0 {
		return 0
	}
	var ds int
	for l.cost > l.costLimit {
		last := l.lruList.Tail()
		if last == nil {
			// No data left to evictOverages. Avoid looping forever.
			return ds
		}
		ds += l.delete(last.Data.key)
	}
	return ds
}

// Delete removes an entry from the cache, returning cost savings.
func (l *Cache[K, V]) Delete(key K) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, ok := l.index[key]
	if !ok {
		return 0
	}

	return l.delete(key)
}

// Set adds a new value to the cache.
// Set may also be used to bump a value to the top of the cache.
func (l *Cache[K, V]) Set(key K, v V, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove existing key if it exists.
	l.delete(key)

	l.cost += l.coster(v)
	l.evictExpires()
	l.evictOverages()

	deadline := time.Now().Add(ttl)
	var deadlineKey string

	// If we're getting insert conflicts, we bump the deadline in an
	// exponentially increasing fashion to prevent thrashing.
	// TODO: We should use a string slice as the value to more cleanly
	// resolve conflicts.
	conflictDelay := time.Nanosecond

	// It's possible that multiple keys have the same deadline, in which case
	// we bump the deadline by a nanosecond.
	for {
		deadlineKey = formatDeadlineKey(deadline)
		_, ok := l.ttlTrie.Get(deadlineKey)
		if !ok {
			break
		}
		deadline = deadline.Add(conflictDelay)
		conflictDelay *= 2
	}
	_, ok := l.ttlTrie.Insert(deadlineKey, key)
	if ok {
		panic(fmt.Sprintf("unexpected update of ttlTrie, cache corrupt: %+v", v))
	}
	l.index[key] = l.lruList.Append(
		dataWithKey[K, V]{
			data:     v,
			key:      key,
			deadline: deadline,
		},
	)
}

func (l *Cache[K, V]) get(key K) (v V, deadline time.Time, exists bool) {
	node, exists := l.index[key]
	if !exists {
		return v, time.Time{}, false
	}
	if time.Now().After(node.Data.deadline) {
		l.delete(key)
		return v, time.Time{}, false
	}

	l.lruList.Pop(node)
	l.index[key] = l.lruList.Append(node.Data)
	return node.Data.data, node.Data.deadline, true
}

// Get retrieves a value from the cache, if it exists.
func (l *Cache[K, V]) Get(key K) (v V, deadline time.Time, exists bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.get(key)
}

// Evict removes all expired entries from the cache.
// Bear in mind Set and Delete will also evict entries, so most users should
// not call Evict directly.
func (l *Cache[K, V]) Evict() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.evictExpires() + l.evictOverages()
}
