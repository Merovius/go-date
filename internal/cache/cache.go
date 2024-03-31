// Copyright 2024 Axel Wagner.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cache implements a very simple random-replacement cache to memoize
// expensive operations.
package cache

import (
	"sync"
)

// DefaultSize is the default size of a cache.
const DefaultSize = 1 << 10

// Cache is a simple random-replacement cache suitable to memoize expensive
// operations.
//
// Its zero value is safe to use. It is safe for concurrent use.
type Cache[K comparable, V any] struct {
	// MaxSize is the maximum size of the cache. If it is zero, DefaultSize is used.
	//
	// If V implements Sizer, it is used to estimate size. Otherwise every
	// element is assumed to have size 1.
	//
	// MaxSize is not safe to mutate concurrently with calls to Get.
	MaxSize int64

	mu sync.RWMutex
	m  map[K]V
	n  int64
}

// Get the element associated with k from the cache, using fill to populate
// missing elements.
func (c *Cache[K, V]) Get(k K, fill func(K) V) V {
	c.mu.RLock()
	if v, ok := c.m[k]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	nv := fill(k)

	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.m[k]; ok {
		// another goroutine filled the cache in the meantime
		return v
	}
	if c.m == nil {
		c.m = make(map[K]V)
	}
	c.m[k] = nv
	c.n += size(nv)
	for k := range c.m {
		if !c.fullRLocked() {
			break
		}
		c.evictLocked(k)
	}
	return nv
}

// full returns whether c is full. c.mu must be held for reading when calling
// it.
func (c *Cache[K, V]) fullRLocked() bool {
	m := c.MaxSize
	if m == 0 {
		m = DefaultSize
	}
	return c.n > m
}

// Evict the element for k from the cache. If there is no such element, Evict
// is a no-op.
func (c *Cache[K, V]) Evict(k K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictLocked(k)
}

// evictLocked evicts the given key from the cache. c.mu must be held for
// writing when calling it.
func (c *Cache[K, V]) evictLocked(k K) {
	if v, ok := c.m[k]; ok {
		delete(c.m, k)
		c.n -= size(v)
	}
}

// Flush removes all elements from the cache.
func (c *Cache[K, V]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.m)
	c.n = 0
}

// Sizer is an optional interface for a value to report its own size. The
// reported size must be positive and never change for the same receiver.
type Sizer interface {
	Size() int64
}

func size[V any](v V) int64 {
	if s, ok := any(v).(Sizer); ok {
		return s.Size()
	}
	return 1
}
