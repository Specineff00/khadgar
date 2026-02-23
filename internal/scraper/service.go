package scraper

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"khadgar"

	"github.com/Khan/genqlient/graphql"
)

// FetchCompanies demonstrates the genqlient flow for this scraper.
//
// How this works:
// 1. Schema + operation:
//   - `schema.graphql` is a local stub of the remote API shape.
//   - `queries/GetCompanies.graphql` defines the operation we want.
//
// 2. Code generation:
//   - `go generate ./...` runs genqlient and creates typed Go code in `generated.go`.
//   - That generated function is `khadgar.PersonalisedCompanies(...)`.
//
// 3. Runtime request:
//   - We create an `http.Client` with a custom RoundTripper (`headerTransport`)
//     to inject headers (e.g. User-Agent) that this endpoint expects.
//   - We wrap it with `graphql.NewClient(url, httpClient)`.
//   - We call the generated function with typed variables (offset, limit, companyName).
//
// 4. Typed response:
//   - genqlient unmarshals JSON into typed structs, so we iterate
//     `resp.PersonalisedCompanies` directly (no manual payload/decoder structs).
//
// Note:
//   - The local schema is only for codegen/type-checking.
//   - As long as queried fields/types/nullability match the real API behavior,
//     generated requests/responses will work even if the full server schema is unknown.
func FetchCompanies() {
	url := "https://api.exp.welcometothejungle.com/graphql"

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

	gqlClient := graphql.NewClient(url, httpClient)

	const (
		limit    = 100
		maxPages = 200
	)

	retryConfig := RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   250 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		JitterFrac:  0.2,
	}
	ctx := context.Background()
	all := make([]Company, 0, 2000)
	seen := make(map[string]struct{}) // dedupe key

	for page := range maxPages {

		var resp *khadgar.PersonalisedCompaniesResponse
		err := doWithRetry(ctx, retryConfig, func(ctx context.Context) (statusCode int, err error) {
			r, err := khadgar.PersonalisedCompanies(
				ctx,
				gqlClient,
				nextOffset(page, limit),
				limit,
				"",
			)
			if err != nil {
				return statusCodeFromError(err), err
			}

			resp = r
			return http.StatusOK, nil
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				log.Printf("scrape canceled: %v", err)
				return
			}
			log.Printf("fetch companies page=%d failed after retries: %v", page, err)
			return
		}
		if resp == nil {
			log.Printf("fetch companies page=%d returned nil response", page)
			return
		}

		cCount := len(resp.PersonalisedCompanies)
		if cCount == 0 {
			// No more data
			return
		}

		mappedPage := toCompanies(resp.PersonalisedCompanies)
		all = mergeUnique(all, mappedPage, seen)

		// Last partial page => likely end.
		// So 32 would be considered added and therefore it's time to break out and finish
		if shouldStop(cCount, limit) {
			break
		}
	}

	// for _, comp := range resp.PersonalisedCompanies {
	// 	fmt.Printf("--- \nName: %s\nSize: %s\nDescription: %s\n", comp.Name, comp.Size.Value, comp.ShortDescription)
	// }
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
}
