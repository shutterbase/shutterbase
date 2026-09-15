package web

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// ipLimiter is a per-key token bucket (requests per minute, burst = budget).
// ponytail: per-instance; the bulk-download quota that must be shared across
// replicas is counted in the jobs table instead.
type ipLimiter struct {
	mu      sync.Mutex
	buckets *expirable.LRU[string, *bucket]
	rate    float64 // tokens per second
	burst   float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter(perMinute int) *ipLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return &ipLimiter{buckets: expirable.NewLRU[string, *bucket](8192, nil, 10*time.Minute), rate: float64(perMinute) / 60, burst: float64(perMinute)}
}

func (l *ipLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets.Get(key)
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets.Add(key, b)
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
