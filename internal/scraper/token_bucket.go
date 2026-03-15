package scraper

import (
	"context"
	"sync"
	"time"
)

type TokenBucketLimiter struct {
	mu            sync.Mutex
	hosts         map[string]*bucket
	defaultRate   float64 // tokens per second
	defaultBurst  int     // max tokens
	hostOverrides map[string]hostConfig
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
	rate       float64
	burst      int
}

type hostConfig struct {
	rate  float64
	burst int
}

func NewTokenBucketLimiter(defaultRate float64, defaultBurst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		hosts:         make(map[string]*bucket),
		defaultRate:   defaultRate,
		defaultBurst:  defaultBurst,
		hostOverrides: make(map[string]hostConfig),
	}
}

func (t *TokenBucketLimiter) setHostLimiter(host string, rate float64, burst int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hostOverrides[host] = hostConfig{rate: rate, burst: burst}
}

func (t *TokenBucketLimiter) getBucket(host string) *bucket {
	if b, ok := t.hosts[host]; ok {
		return b
	}
	rate := t.defaultRate
	burst := t.defaultBurst
	if override, ok := t.hostOverrides[host]; ok {
		rate = override.rate
		burst = override.burst
	}

	b := &bucket{
		tokens:     float64(burst),
		lastRefill: time.Now(),
		rate:       rate,
		burst:      burst,
	}
	t.hosts[host] = b
	return b
}

func (t *TokenBucketLimiter) Wait(ctx context.Context, host string) error {
	for {
		t.mu.Lock()
		b := t.getBucket(host)
		now := time.Now()
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * b.rate
		if b.tokens > float64(b.burst) {
			b.tokens = float64(b.burst)
		}
		b.lastRefill = now

		if b.tokens >= 1 {
			b.tokens--
			t.mu.Unlock()
			return nil
		}

		// Wait until time it takes to get one token
		waitSec := (1 - b.tokens) / b.rate
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
