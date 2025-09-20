// Package consistenthash provides consistent hashing implementations for both int64 and string keys
package consistenthash

import (
	"hash"
	"math"
	"sort"
	"strconv"
	"sync"

	"github.com/go-pantheon/fabrica-util/errors"
	"github.com/spaolacci/murmur3"
)

type int64RingNode struct {
	nodeName string
	key      int64
	hash     uint64
}

type int64RingNodes []int64RingNode

func (r int64RingNodes) Len() int           { return len(r) }
func (r int64RingNodes) Less(i, j int) bool { return r[i].hash < r[j].hash }
func (r int64RingNodes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// Int64HashRing is a consistent hash ring for int64 keys.
type Int64HashRing struct {
	sync.RWMutex
	virtualSpots int
	nodes        int64RingNodes
	hashCache    sync.Pool
}

// NewInt64Ring creates a new Int64HashRing with the given number of virtual spots.
func NewInt64Ring(virtualSpots int) *Int64HashRing {
	if virtualSpots <= 0 {
		virtualSpots = DefaultVirtualSpots
	}

	return &Int64HashRing{
		virtualSpots: virtualSpots,
		hashCache: sync.Pool{
			New: func() any {
				return murmur3.New64()
			},
		},
	}
}

// AddNode adds a new node to the int64 hash ring with the specified node name
// It creates virtual nodes based on the configured virtual spots and returns error if any
func (h *Int64HashRing) AddNode(nodeName string) (err error) {
	h.Lock()
	defer h.Unlock()

	hasher := h.hashCache.Get().(hash.Hash64)
	defer h.hashCache.Put(hasher)

	nodes := make(int64RingNodes, 0, h.virtualSpots)

	for i := range h.virtualSpots {
		keyStr := nodeName + ":" + strconv.Itoa(i)

		hasher.Reset()
		_, err = hasher.Write([]byte(keyStr))

		if err != nil {
			return errors.Wrap(err, "write to hasher failed")
		}

		hash64 := hasher.Sum64()

		nodes = append(nodes, int64RingNode{
			nodeName: nodeName,
			key:      convertToInt64(hash64),
			hash:     hash64,
		})
	}

	h.nodes = append(h.nodes, nodes...)
	sort.Sort(h.nodes)

	return nil
}

// RemoveNode removes a node from the hash ring.
func (h *Int64HashRing) RemoveNode(nodeName string) {
	h.Lock()
	defer h.Unlock()

	filtered := h.nodes[:0]

	for _, n := range h.nodes {
		if n.nodeName != nodeName {
			filtered = append(filtered, n)
		}
	}

	h.nodes = filtered
}

// GetNode returns the node name for the given key
// It finds the closest virtual node in the ring and returns its node name
// Also returns a boolean indicating if a node was found
func (h *Int64HashRing) GetNode(key int64) (nodeName string, ok bool) {
	h.RLock()
	defer h.RUnlock()

	if len(h.nodes) == 0 {
		return "", false
	}

	var targetHash uint64
	if key >= 0 {
		targetHash = uint64(key)
	} else {
		targetHash = ^uint64(0) - uint64(-key) + 1 // Map negative numbers to the high range to keep hash distribution uniform
	}

	idx := sort.Search(len(h.nodes), func(i int) bool {
		return h.nodes[i].hash >= targetHash
	})

	if idx == len(h.nodes) {
		idx = 0
	}

	return h.nodes[idx].nodeName, true
}

// convertToInt64 safely converts uint64 to int64, handling possible overflow
func convertToInt64(val uint64) int64 {
	if val > uint64(math.MaxInt64) {
		return math.MaxInt64
	}

	return int64(val)
}

func (h *Int64HashRing) Len() int {
	h.RLock()
	defer h.RUnlock()

	return len(h.nodes)
}
