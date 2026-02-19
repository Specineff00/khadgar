package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

	all := make([]Company, 0, 2000)
	seen := make(map[string]struct{}) // dedupe key

	for page := range maxPages {

		resp, err := khadgar.PersonalisedCompanies(
			context.Background(),
			gqlClient,
			nextOffset(page, limit),
			limit,
			"",
		)
		if err != nil {
			fmt.Printf("Some error %s", err)
			return
		}

		cCount := len(resp.PersonalisedCompanies)
		if cCount == 0 {
			// No more data
			return
		}

		for _, c := range resp.PersonalisedCompanies {
			// Prefer stable ID if available. Name fallback is imperfect.
			key := strings.ToLower(strings.TrimSpace(c.Name))

			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}

			all = append(all, Company{
				Name:             c.Name,
				ShortDescription: c.ShortDescription,
				Size:             c.Size.Value,
			})
		}

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
