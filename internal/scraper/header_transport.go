package scraper

import (
	"fmt"
	"net/http"
)

type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	for k, v := range t.headers {
		clone.Header.Set(k, v)
	}

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(clone)
	if err != nil {
		return nil, err
	}

	// If the meta key exists then try get the value
	if meta, ok := clone.Context().Value(responseMetaKey{}).(*ResponseMeta); ok {
		meta.RetryAfter = resp.Header.Get("Retry-After")
		fmt.Printf("retry after header: %s", meta.RetryAfter)
	}
	return resp, nil
}

type responseMetaKey struct{}

type ResponseMeta struct {
	RetryAfter string
}
