package cache

import (
	"sync"
	"time"
)

type Cache struct {
	ttl    time.Duration
	cache  map[string]CacheItem
	mu     sync.RWMutex
	ticker *time.Ticker
	done   chan struct{}
}

type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}

	c := &Cache{
		ttl:    ttl,
		cache:  make(map[string]CacheItem),
		ticker: time.NewTicker(ttl),
		done:   make(chan struct{}),
	}

	go c.cleanupLoop()
	return c
}

func (c *Cache) cleanupLoop() {
	for {
		select {
		case <-c.ticker.C:
			c.Clear()
		case <-c.done:
			c.ticker.Stop()
			return
		}
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		delete(c.cache, key)
		return nil, false
	}

	return item.Value, true
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.cache[key] = CacheItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.cache {
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

func (c *Cache) Close() {
	close(c.done)
}
