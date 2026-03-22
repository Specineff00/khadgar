package scraper

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"khadgar/internal/scraper/mock"
)

var errCannotMarshal = errors.New("cannot unmarshal")

type mockBody struct {
	Name string `json:"name"`
}

type unmarshalable struct{}

func TestNewGraphClient_ReturnsNonNullClient(t *testing.T) {
	c := NewGraphQLClient("http://graphql.com")
	if c == nil {
		t.Fatal("expected non nill graphql.com")
	}
}

func TestNewRESTClient_ReturnsNonNullClient(t *testing.T) {
	c := NewRESTClient()
	wantTimeOut := 20 * time.Second
	if c.Timeout != wantTimeOut {
		t.Fatal("expected 20 second timeout")
	}
	if c == nil {
		t.Fatal("expected a non nill RestClient")
	}
}

func TestDoRequest_InvalidBodyReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient, http.MethodGet,
		"http://example.com",
		unmarshalable{},
		site,
		company,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatal("expected resp to be nil, got present")
	}

	if !errors.Is(err, errCannotMarshal) {
		t.Errorf("expected error to wrap errCannotMarshal, got %v", err)
	}
}

func TestDoRequest_InvalidMethodReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		"GET ",
		"http://example.com",
		mockBody{Name: "Nik"},
		site,
		company,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatal("expected nil response on error")
	}

	if !strings.Contains(err.Error(), "request lever acme") {
		t.Errorf("expected siteRequestError, %v", err.Error())
	}

	if !strings.Contains(err.Error(), "invalid method") {
		t.Errorf("expected invalid method, got %v", err.Error())
	}
}

func TestDoRequest_NoBodyInvalidURLReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"//oo\\x00",
		nil,
		site,
		company,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatal("expected nil response on error")
	}

	if !strings.Contains(err.Error(), "request lever acme") {
		t.Errorf("expected siteRequestError, %v", err.Error())
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected invalid parse, got %v", err.Error())
	}
}

func TestDoRequest_NoBodyInvalidMethodReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		"GET ",
		"http://example.com",
		nil,
		site,
		company,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatal("expected nil response on error")
	}

	if !strings.Contains(err.Error(), "request lever acme") {
		t.Errorf("expected siteRequestError, %v", err.Error())
	}

	if !strings.Contains(err.Error(), "invalid method") {
		t.Errorf("expected invalid method, got %v", err.Error())
	}
}

func TestDoRequest_InvalidURLReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"//oo\\x00",
		mockBody{Name: "Nik"},
		site,
		company,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatal("expected nil response on error")
	}

	if !strings.Contains(err.Error(), "request lever acme") {
		t.Errorf("expected siteRequestError, %v", err.Error())
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected invalid parse, got %v", err.Error())
	}
}

func TestDoRequest_OKResponseNilErrorReturnsResponse(t *testing.T) {
	wantBody := `"ok": true`
	httpClient := &http.Client{
		Transport: mock.Transport(),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"http://example.com",
		mockBody{Name: "Nik"},
		site,
		company,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp == nil {
		t.Fatalf("expected response")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("invalid read of body, %v", err)
		return
	}

	if string(body) != wantBody {
		t.Errorf("incorrect body, got %v want %v", body, wantBody)
	}
}

func TestDoRequest_ResponseNonRetryableErrorReturnsError(t *testing.T) {
	wantError := context.Canceled
	httpClient := &http.Client{
		Transport: mock.Transport(mock.Error(wantError)),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"http://example.com",
		mockBody{Name: "Nik"},
		site,
		company,
	)

	if resp != nil {
		t.Fatal("expected nil response")
	}

	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, wantError) {
		t.Errorf("incorrect error, got %v want %v", err, wantError)
	}
}

func TestDoRequest_ResponseRetryableErrorReturnsError(t *testing.T) {
	wantError := syscall.ECONNABORTED
	httpClient := &http.Client{
		Transport: mock.Transport(mock.Error(wantError)),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"http://example.com",
		mockBody{Name: "Nik"},
		site,
		company,
	)

	if resp != nil {
		t.Fatal("expected nil response")
	}

	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, wantError) {
		t.Errorf("incorrect error, got %v want %v", err, wantError)
	}
}

func TestDoRequest_StatusCode300ReturnsError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mock.Transport(mock.StatusCode(300)),
	}
	site := "lever"
	company := "acme"
	resp, err := doRequest(
		context.Background(),
		httpClient,
		http.MethodGet,
		"http://example.com",
		mockBody{Name: "Nik"},
		site,
		company,
	)

	if resp != nil {
		t.Fatal("expected nil response")
	}

	if err == nil {
		t.Fatalf("expected error")
	}

	wantCode := "300"

	if !strings.Contains(err.Error(), wantCode) {
		t.Errorf("incorrect status code, got %v want %v", err.Error(), wantCode)
	}
}
