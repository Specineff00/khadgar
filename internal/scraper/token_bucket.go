package scraper

import (
	"context"
	"sync"
	"time"
)

type TokenBucketLimiter struct {
	mu    sync.Mutex
	hosts map[string]*bucket
	rate  float64 // tokens per second
	burst int     // max tokens
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		hosts: make(map[string]*bucket),
		rate:  rate,
		burst: burst,
	}
}

func (t *TokenBucketLimiter) getBucket(host string) *bucket {
	if b, ok := t.hosts[host]; ok {
		return b
	}
	b := &bucket{tokens: float64(t.burst), lastRefill: time.Now()}
	t.hosts[host] = b
	return b
}

func (t *TokenBucketLimiter) Wait(ctx context.Context, host string) error {
	for {
		t.mu.Lock()
		b := t.getBucket(host)
		now := time.Now()
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = min(elapsed*t.rate, float64(t.burst))
		b.lastRefill = now

		if b.tokens >= 1 {
			b.tokens--
			t.mu.Unlock()
			return nil
		}

		// Wait until time it takes to get one token
		waitSec := (1 - b.tokens) / t.rate
		wait := time.Duration(waitSec * float64(time.Second))
		t.mu.Unlock()

		// Either wait or return if ctx is cancelled
		select {
		case <-time.After(wait):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
