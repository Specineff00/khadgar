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
)

type mockBody struct {
	Name string `json:"name"`
}

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
		Transport: &mockTransport{},
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
		Transport: MockTransport.New(),
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
		Transport: MockTransport.New(),
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
		Transport: MockTransport.New(),
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
		Transport: MockTransport.New(),
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
		Transport: MockTransport.New(),
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
		Transport: MockTransport.WithError(wantError),
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
		Transport: MockTransport.WithError(wantError),
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
		Transport: MockTransport.WithStatusCode(300),
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

//
// func doRequest(
// 	ctx context.Context,
// 	httpClient *http.Client,
// 	method string,
// 	url string,
// 	payload any,
// 	site string,
// 	company string,
// ) (*http.Response, error) {
// 	retryError := func(err error) error {
// 		return siteCompanyRetryError(site, company, err)
// 	}
//
// 	var request *http.Request
// 	if payload != nil {
// 		body, err := json.Marshal(payload)
// 		if err != nil {
// 			return nil, siteMarshalError(site, company, err)
// 		}
// 		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
// 		if err != nil {
// 			return nil, siteRequestError(site, company, err)
// 		}
// 		request = req
// 	} else {
// 		req, err := http.NewRequestWithContext(ctx, method, url, nil)
// 		if err != nil {
// 			return nil, siteRequestError(site, company, err)
// 		}
// 		request = req
// 	}
//
// 	resp, err := httpClient.Do(request)
// 	if err != nil {
// 		if resp != nil && resp.Body != nil {
// 			resp.Body.Close()
// 		}
// 		if isRetryable(err, 0) {
// 			return nil, retryError(err)
// 		}
// 		return nil, fmt.Errorf("%s %s: %w", site, company, err)
// 	}
//
// 	if resp.StatusCode != http.StatusOK {
// 		io.Copy(io.Discard, resp.Body)
// 		resp.Body.Close()
// 		return nil, checkSiteStatusError(site, company, resp.StatusCode)
// 	}
//
// 	return resp, nil
// }
//
