package cache

type Coster[T any] func(v T) int

// ConstantCost always returns 1.
func ConstantCost[T any](_ T) int {
	return 1
}

// LRU implements a thread-safe, least-frequently-used cache structure.
// When the cache exceeds a given cost parameter, the oldest chunks of data are discarded.
type LRU[NodeType any] struct {
	index map[string]NodeType
	// coster allows for user-defined relative weighting of cache members.
	coster Coster[NodeType]
	// maxCost sets the maximum
	maxCost int64
}

func NewLRU(cost Coster, maxCost int64) *LRU {

}
