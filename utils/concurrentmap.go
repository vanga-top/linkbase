package utils

import (
	"go.uber.org/atomic"
	"sync"
)

type ConcurrentMap[K comparable, V any] struct {
	inner sync.Map
	// Self-managed Len(), see: https://github.com/golang/go/issues/20680.
	len atomic.Uint64
}

// Insert inserts the key-value pair to the concurrent map
func (m *ConcurrentMap[K, V]) Insert(key K, value V) {
	_, loaded := m.inner.LoadOrStore(key, value)
	if !loaded {
		m.len.Inc()
	} else {
		m.inner.Store(key, value)
	}
}
