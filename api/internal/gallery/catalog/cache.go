package catalog

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"
)

const (
	minTTL = 30 * time.Second
	maxTTL = 10 * time.Minute
)

// cache is a TTL LRU with request coalescing: a cold key under load runs the
// loader once, every concurrent caller waits for that result. Published data
// is mutable, so TTL is the freshness guarantee (clamped to 30s..10m).
type cache struct {
	lru *expirable.LRU[string, any]
	sf  singleflight.Group
	ttl time.Duration
}

func clampTTL(ttl time.Duration) time.Duration {
	if ttl < minTTL {
		return minTTL
	}
	if ttl > maxTTL {
		return maxTTL
	}
	return ttl
}

func newCache(ttl time.Duration, maxEntries int) *cache {
	ttl = clampTTL(ttl)
	if maxEntries <= 0 {
		maxEntries = 5000
	}
	return &cache{lru: expirable.NewLRU[string, any](maxEntries, nil, ttl), ttl: ttl}
}

func (c *cache) get(key string, load func() (any, error)) (any, error) {
	if v, ok := c.lru.Get(key); ok {
		return v, nil
	}
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if v, ok := c.lru.Get(key); ok {
			return v, nil
		}
		v, err := load()
		if err != nil {
			return nil, err
		}
		c.lru.Add(key, v)
		return v, nil
	})
	return v, err
}

// cached is the typed helper around cache.get.
func cached[T any](c *cache, key string, load func() (T, error)) (T, error) {
	v, err := c.get(key, func() (any, error) { return load() })
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}

func (c *cache) purge()   { c.lru.Purge() }
func (c *cache) len() int { return c.lru.Len() }
