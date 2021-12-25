package cache

import "github.com/ammario/cache/internal/doublelist"

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
	index map[string]*doublelist.Node[dataWithKey[NodeType]]
	list  *doublelist.List[dataWithKey[NodeType]]
	// coster allows for user-defined relative weighting of cache members.
	coster Coster[NodeType]
	cost   int
	// maxCost sets the maximum
	maxCost int
}

// NewLRU instantiates a ready-to-use LRU cache.
func NewLRU[NodeType any](cost Coster[NodeType], maxCost int) *LRU[NodeType] {
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

// Delete deletes a value from the cache.
func (l *LRU[T]) Delete(key string) {
	node, ok := l.index[key]
	if !ok {
		return
	}
	l.list.Pop(node)
	l.cost -= l.coster(node.Data.data)
	delete(l.index, key)
}

// Set adds a new value to the cache.
// Set should be used to add new values and reset values to the top of the cache.
func (l *LRU[T]) Set(key string, v T) {
	l.Delete(key)
	l.cost += l.coster(v)
	l.evictOverages()
	l.index[key] = l.list.Append(dataWithKey[T]{data: v, key: key})
}

// Get retrieves a value from the cache, if it exists.
func (l *LRU[T]) Get(key string) (v T, exists bool) {
	node, exists := l.index[key]
	if !exists {
		return v, false
	}
	l.list.Pop(node)
	l.index[key] = l.list.Append(node.Data)
	return node.Data.data, true
}
