package lru

import (
	"sync"

	"github.com/ammario/lru-cache/internal/doublelist"
)

type Coster[T any] func(v T) int

// ConstantCost always returns 1.
func ConstantCost[T any](_ T) int {
	return 1
}

// dataWithKey bundles data with its reference key.
// This structure allows for reverse lookup from the doubly-linked list to the index.
type dataWithKey[T any] struct {
	data T
	key  string
}

// LRU implements a least-frequently-used cache structure.
// When the cache exceeds a given cost limit, the oldest chunks of data are discarded.
type LRU[NodeType any] struct {
	mu sync.Mutex

	index map[string]*doublelist.Node[dataWithKey[NodeType]]
	list  *doublelist.List[dataWithKey[NodeType]]
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
		list:    &doublelist.List[dataWithKey[NodeType]]{},
		coster:  cost,
		maxCost: maxCost,
	}
}

func (l *LRU[T]) evictOverages() {
	for l.cost > l.maxCost {
		exit, ok := l.list.PopTail()
		if !ok {
			// No data left to evictOverages. Avoid looping forever.
			return
		}
		l.cost -= l.coster(exit.Data.data)
		delete(l.index, exit.Data.key)
	}
}

func (l *LRU[T]) delete(key string) {
	node, ok := l.index[key]
	if !ok {
		return
	}
	l.list.Pop(node)
	l.cost -= l.coster(node.Data.data)
	delete(l.index, key)
}

// Delete removes an entry from the cache.
func (l *LRU[T]) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.delete(key)
}

// Set adds a new value to the cache.
// Set may also be used to bump a value to the top of the cache.
func (l *LRU[T]) Set(key string, v T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.delete(key)
	l.cost += l.coster(v)
	l.evictOverages()
	l.index[key] = l.list.Append(dataWithKey[T]{data: v, key: key})
}

// Get retrieves a value from the cache, if it exists.
func (l *LRU[T]) Get(key string) (v T, exists bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, exists := l.index[key]
	if !exists {
		return v, false
	}
	l.list.Pop(node)
	l.index[key] = l.list.Append(node.Data)
	return node.Data.data, true
}
