package scraper

import (
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
