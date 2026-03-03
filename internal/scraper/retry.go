package scraper

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"strconv"
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

func (s *Service) doWithRetry(ctx context.Context, fn func(context.Context) (statusCode int, err error)) error {
	if s.RetryConfig.MaxAttempts < 1 {
		s.RetryConfig.MaxAttempts = 1
	}

	if s.RetryConfig.BaseDelay <= 0 {
		s.RetryConfig.BaseDelay = 250 * time.Millisecond
	}

	if s.RetryConfig.MaxDelay < s.RetryConfig.BaseDelay {
		s.RetryConfig.MaxDelay = s.RetryConfig.BaseDelay
	}

	var lastErr error

	for attempt := range s.RetryConfig.MaxAttempts {
		statusCode, err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		retryable := isRetryable(err, statusCode)
		s.logFailedAttempt(attempt, statusCode, retryable, err)
		if !retryable || attempt == s.RetryConfig.MaxAttempts-1 {
			s.logRetrysExhausted(attempt, statusCode, err)
			break
		}

		delay := nextBackoff(
			attempt,
			s.RetryConfig.BaseDelay,
			s.RetryConfig.MaxDelay,
			s.RetryConfig.JitterFrac,
		)
		// Only change delay if we can get the retry after from the response's header
		if statusCode == 429 {
			if meta, ok := ctx.Value(responseMetaKey{}).(ResponseMeta); ok && meta.RetryAfter != "" {
				s.Logger.Info("something in retry", "meta", meta.RetryAfter)
				if secs, err := strconv.Atoi(meta.RetryAfter); err == nil {
					delay = time.Duration(secs)
					s.logRetryAfter(delay)
				}
			}
		}

		s.logBackoff(attempt, delay)

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

func (s *Service) logFailedAttempt(attemptNum, statusCode int, retryable bool, err error) {
	s.Logger.Warn("attempt failed",
		"attempt", attemptNum,
		"max_attempts", s.RetryConfig.MaxAttempts,
		"status_code", statusCode,
		"retryable", retryable,
		"err", err,
	)
}

func (s *Service) logRetrysExhausted(attemptNum, statusCode int, err error) {
	s.Logger.Error("retries exhausted",
		"attempt", attemptNum,
		"status_code", statusCode,
		"err", err,
	)
}

func (s *Service) logBackoff(attemptNum int, delay time.Duration) {
	s.Logger.Info("retrying after backoff",
		"attempt", attemptNum+1,
		"delay_ms", delay.Milliseconds(),
	)
}

func (s *Service) logRetryAfter(delay time.Duration) {
	s.Logger.Info("retrying after delay from header",
		"delay_ms", delay.Milliseconds(),
	)
}
