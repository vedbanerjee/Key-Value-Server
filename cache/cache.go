package cache

import (
	"container/list"
)

type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	order    *list.List
}

type cacheEntry struct {
	key   string
	value string
}

// NewLRUCache creates a new LRU cache with a given capacity.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a value from the cache and moves the item to the front (most recently used).
func (c *LRUCache) Get(key string) (string, bool) {
	if elem, found := c.cache[key]; found {
		// Move the accessed item to the front to mark it as recently used
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheEntry).value, true
	}
	return "", false
}

// Put inserts a key-value pair into the cache, evicting the least recently used item if necessary.
func (c *LRUCache) Put(key, value string) {
	// If the key is already in the cache, update it and move it to the front
	if elem, found := c.cache[key]; found {
		c.order.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	// If the cache has reached its capacity, remove the least recently used element
	if c.order.Len() >= c.capacity {
		c.evict()
	}

	// Add the new entry to the front of the list
	entry := &cacheEntry{key: key, value: value}
	elem := c.order.PushFront(entry)
	c.cache[key] = elem
}

// evict removes the least recently used item (the item at the back of the list).
func (c *LRUCache) evict() {
	// Remove the item at the back of the list (least recently used)
	elem := c.order.Back()
	if elem != nil {
		c.order.Remove(elem)
		delete(c.cache, elem.Value.(*cacheEntry).key)
	}
}

// DeleteKey removes the given key from the cache.
func (c *LRUCache) DeleteKey(key string) {
	if elem, found := c.cache[key]; found {
		c.order.Remove(elem)
		delete(c.cache, key)
	}
}
