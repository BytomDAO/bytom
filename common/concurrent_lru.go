package common

import (
	"sync"

	"github.com/golang/groupcache/lru"
)

// Cache is an LRU cache. It is safe for concurrent access.
type Cache struct {
	cache *lru.Cache
	sync.RWMutex
}

// NewCache creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func NewCache(maxEntries int) *Cache {
	return &Cache{cache: lru.New(maxEntries)}
}

// Add adds a value to the cache.
func (c *Cache) Add(key, value interface{}) {
	c.Lock()
	defer c.Unlock()
	c.cache.Add(key, value)
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	c.Lock()
	defer c.Unlock()
	return c.cache.Get(key)
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) {
	c.Lock()
	defer c.Unlock()
	c.cache.Remove(key)
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	c.Lock()
	defer c.Unlock()
	c.cache.RemoveOldest()
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.RLock()
	defer c.RUnlock()
	return c.cache.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.cache.Clear()
}
