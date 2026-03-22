package scraper

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

var MockTransport = &MockTransportFactory{}

type MockTransportFactory struct{}

var errCannotMarshal = errors.New("cannot not marshal")

type mockTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTrip == nil {
		return nil, errors.New("mock transport: roundTrip not configured")
	}
	return m.roundTrip(req)
}

func (*MockTransportFactory) New() *mockTransport {
	transport := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`"ok": true`)),
		}, nil
	}
	return &mockTransport{
		roundTrip: transport,
	}
}

func (*MockTransportFactory) WithError(err error) *mockTransport {
	transport := func(r *http.Request) (*http.Response, error) {
		return nil, err
	}
	return &mockTransport{
		roundTrip: transport,
	}
}

func (*MockTransportFactory) WithStatusCode(code int) *mockTransport {
	transport := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(`"ok": true`)),
		}, nil
	}
	return &mockTransport{
		roundTrip: transport,
	}
}

type unmarshalable struct{}

func (unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errCannotMarshal
}
