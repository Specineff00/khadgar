package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

func NewGraphQLClient(url string) graphql.Client {
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		Transport: headerTransport{
			base: http.DefaultTransport,
			headers: map[string]string{
				"User-Agent":   "Mozilla/5.0...",
				"Content-Type": "application/json",
			},
		},
	}
	return graphql.NewClient(url, httpClient)
}

func NewRESTClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		Transport: headerTransport{
			base: http.DefaultTransport,
			headers: map[string]string{
				"User-Agent":   "Mozilla/5.0...",
				"Content-Type": "application/json",
			},
		},
	}
}

func doRequest(
	ctx context.Context,
	httpClient *http.Client,
	method string,
	url string,
	payload any,
	site string,
	company string,
) (*http.Response, error) {
	retryError := func(err error) error {
		return siteCompanyRetryError(site, company, err)
	}

	var request *http.Request
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, siteMarshalError(site, company, err)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, siteRequestError(site, company, err)
		}
		request = req
	} else {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, siteRequestError(site, company, err)
		}
		request = req
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if isRetryable(err, 0) {
			return nil, retryError(err)
		}
		return nil, fmt.Errorf("%s %s: %w", site, company, err)
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, checkSiteStatusError(site, company, resp.StatusCode)
	}

	return resp, nil
}
