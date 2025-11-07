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

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *LRUCache) Get(key string) (string, bool) {
	if elem, found := c.cache[key]; found {
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheEntry).value, true
	}
	return "", false
}

func (c *LRUCache) Put(key, value string) {
	if elem, found := c.cache[key]; found {
		c.order.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	if c.order.Len() >= c.capacity {
		c.evict()
	}

	entry := &cacheEntry{key: key, value: value}
	elem := c.order.PushFront(entry)
	c.cache[key] = elem
}

func (c *LRUCache) evict() {
	elem := c.order.Back()
	if elem != nil {
		c.order.Remove(elem)
		delete(c.cache, elem.Value.(*cacheEntry).key)
	}
}

func (c *LRUCache) DeleteKey(key string) {
	if elem, found := c.cache[key]; found {
		c.order.Remove(elem)
		delete(c.cache, key)
	}
}
