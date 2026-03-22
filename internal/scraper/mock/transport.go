package mock

import (
	"io"
	"net/http"
	"strings"
)

type TransportOption func(*transportConfig)

type transportConfig struct {
	statusCode int
	body       string
	err        error
}

func StatusCode(n int) TransportOption {
	return func(tc *transportConfig) { tc.statusCode = n }
}

func Body(s string) TransportOption {
	return func(tc *transportConfig) { tc.body = s }
}

func Error(e error) TransportOption {
	return func(tc *transportConfig) { tc.err = e }
}

func Transport(opts ...TransportOption) http.RoundTripper {
	tc := &transportConfig{statusCode: 200}

	for _, opt := range opts {
		opt(tc)
	}
	return &mockTransport{config: tc}
}

type mockTransport struct {
	config *transportConfig
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if m.config != nil {
		return nil, m.config.err
	}
	return &http.Response{
		StatusCode: m.config.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.config.body)),
	}, nil
}
