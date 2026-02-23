package scraper

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestRetryStatusCodes_ValidCodes(t *testing.T) {
	table := []struct {
		name string
		code int
		want bool
	}{
		{"429", 429, true},
		{"500", 500, true},
		{"502", 502, true},
		{"503", 503, true},
		{"504", 504, true},
		{"505", 505, false},
		{"200", 200, false},
		{"1000", 1000, false},
	}
	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryableStatus(tc.code)
			if got != tc.want {
				t.Errorf(
					"isRetryableStatus(%v) = %v, want %v",
					tc.code,
					got,
					tc.want,
				)
			}
		})
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		want       bool
	}{
		{
			name: "context canceled is not retryable",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "network timeout error is retryable",
			err:  timeoutErr{},
			want: true,
		},
		{
			name:       "retryable status is used when error present",
			err:        errors.New("http failure"),
			statusCode: 503,
			want:       true,
		},
		{
			name:       "non-retryable status is used when error present",
			err:        errors.New("http failure"),
			statusCode: 404,
			want:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryable(tc.err, tc.statusCode)
			if got != tc.want {
				t.Fatalf("isRetryable(%v, %d) = %v, want %v", tc.err, tc.statusCode, got, tc.want)
			}
		})
	}
}

var _ net.Error = timeoutErr{}
