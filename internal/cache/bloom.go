package cache

import (
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	// defaultExpectedItems is the estimated number of unique short codes.
	defaultExpectedItems = 1_000_000

	// defaultFalsePositiveRate is the target false-positive probability.
	defaultFalsePositiveRate = 0.01
)

// BloomFilter provides a probabilistic filter to defend against cache penetration.
// If a short code is definitely NOT in the bloom filter, we skip both cache and DB lookups.
// False positives are acceptable (they just fall through to normal cache-aside flow).
type BloomFilter struct {
	mu     sync.RWMutex
	filter *bloom.BloomFilter
}

// NewBloomFilter creates a bloom filter sized for the expected number of items.
func NewBloomFilter(expectedItems uint, fpRate float64) *BloomFilter {
	if expectedItems == 0 {
		expectedItems = defaultExpectedItems
	}
	if fpRate <= 0 || fpRate >= 1 {
		fpRate = defaultFalsePositiveRate
	}

	return &BloomFilter{
		filter: bloom.NewWithEstimates(expectedItems, fpRate),
	}
}

// Add registers a short code in the bloom filter.
func (bf *BloomFilter) Add(code string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	bf.filter.AddString(code)
}

// MayExist returns true if the code might exist (could be a false positive).
// Returns false if the code is definitely not in the set.
func (bf *BloomFilter) MayExist(code string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.filter.TestString(code)
}

// AddBatch registers multiple short codes.
func (bf *BloomFilter) AddBatch(codes []string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	for _, code := range codes {
		bf.filter.AddString(code)
	}
}

// ApproximateCount returns the estimated number of items in the filter.
func (bf *BloomFilter) ApproximateCount() uint32 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.filter.ApproximatedSize()
}
