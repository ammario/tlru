package tlru

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/ammario/tlru/internal/doublelist"
	"github.com/armon/go-radix"
)

type Coster[T any] func(v T) int

// ConstantCost always returns 1.
func ConstantCost[T any](_ T) int {
	return 1
}

// dataWithKey bundles data with its reference key.
// This structure allows for reverse lookup from the doubly-linked list to the index.
type dataWithKey[T any] struct {
	data     T
	key      string
	deadline time.Time
}

// LRU implements a least-frequently-used cache structure.
// When the cache exceeds a given cost limit, the oldest chunks of data are discarded.
type LRU[NodeType any] struct {
	mu sync.Mutex

	index map[string]*doublelist.Node[dataWithKey[NodeType]]
	// lruList contains entries in order of least-recently-used to most-recently-used.
	lruList *doublelist.List[dataWithKey[NodeType]]
	// ttlTrie contains entries in order of expires first to expires last.
	// Entries are sorted by their UnixNano deadline.
	ttlTrie *radix.Tree
	// coster allows for user-defined relative weighting of cache members.
	coster Coster[NodeType]
	cost   int
	// maxCost sets the maximum
	maxCost int
}

// New instantiates a ready-to-use LRU cache. It is safe for concurrent use.
func New[NodeType any](cost Coster[NodeType], maxCost int) *LRU[NodeType] {
	return &LRU[NodeType]{
		index:   make(map[string]*doublelist.Node[dataWithKey[NodeType]]),
		lruList: &doublelist.List[dataWithKey[NodeType]]{},
		ttlTrie: radix.New(),
		coster:  cost,
		maxCost: maxCost,
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

func (l *LRU[T]) delete(key string) int {
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

func (l *LRU[T]) evictExpires() int {
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

		k := v.(string)
		ds += l.delete(k)
	}
}

func (l *LRU[T]) evictOverages() int {
	var ds int
	for l.cost > l.maxCost {
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
func (l *LRU[T]) Delete(key string) int {
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
func (l *LRU[T]) Set(key string, v T, ttl time.Duration) {
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
		dataWithKey[T]{
			data:     v,
			key:      key,
			deadline: deadline,
		},
	)
}

// Get retrieves a value from the cache, if it exists.
func (l *LRU[T]) Get(key string) (v T, deadline time.Time, exists bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

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

// Evict removes all expired entries from the cache.
func (l *LRU[T]) Evict() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.evictExpires() + l.evictOverages()
}
