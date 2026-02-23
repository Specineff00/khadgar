package scraper

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"syscall"
	"time"

	gql "github.com/Khan/genqlient/graphql"
)

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// TODO:
// Fully test
func isRetryable(err error, statusCode int) bool {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}

		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if errors.Is(urlErr.Err, context.Canceled) || errors.Is(urlErr.Err, context.DeadlineExceeded) {
				return false
			}
			if ne, ok := urlErr.Err.(net.Error); ok && ne.Timeout() {
				return true
			}
		}

		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return true
		}

		if errors.Is(err, syscall.ECONNABORTED) ||
			errors.Is(err, syscall.ECONNRESET) ||
			errors.Is(err, syscall.ECONNREFUSED) ||
			errors.Is(err, syscall.ETIMEDOUT) {
			return true
		}
	}
	if statusCode != 0 {
		return isRetryableStatus(statusCode)
	}
	return false
}

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	JitterFrac  float64
}

func doWithRetry(ctx context.Context, cfg RetryConfig, fn func(context.Context) (statusCode int, err error)) error {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}

	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 250 * time.Millisecond
	}

	if cfg.MaxDelay < cfg.BaseDelay {
		cfg.MaxDelay = cfg.BaseDelay
	}

	var lastErr error

	for attempt := range cfg.MaxAttempts {
		statusCode, err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		if !isRetryable(err, statusCode) || attempt == cfg.MaxAttempts-1 {
			break
		}

		delay := nextBackoff(attempt, cfg.BaseDelay, cfg.MaxDelay, cfg.JitterFrac)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Try next attempt
		}
	}

	return lastErr
}

func nextBackoff(attempt int, base, max time.Duration, jitterFrac float64) time.Duration {
	delay := base << time.Duration(attempt)

	if delay > max || delay < 0 {
		delay = max
	}

	// jitten in range [1-JitterFrac, 1+JitterFrac]
	if jitterFrac > 0 {
		f := 1 + (rand.Float64()*2-1)*jitterFrac
		delay = time.Duration(float64(delay) * f)
		if delay < 0 {
			delay = 0
		}
	}

	return delay
}

func statusCodeFromError(err error) int {
	if err == nil {
		return 0
	}

	var httpErr *gql.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode
	}

	return 0
}
