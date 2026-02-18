package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"

	"khadgar"

	"github.com/Khan/genqlient/graphql"
)

//go:generate go run github.com/Khan/genqlient ../../genqlient.yaml
func main() {
	// TODO:
	// Pitstop 1: Print what you have
	// Figure out how to deal with pagination and see if pages can be loaded concurrently
	// If concurrency, then learn how to utilise
	// Figure out way to display all: Bubbletea window? output to file?

	// scrapeCompanyName()
	pingGraphQLEndpoint()
}

// pingGraphQLEndpoint demonstrates the genqlient flow for this scraper.
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
func pingGraphQLEndpoint() {
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

	resp, err := khadgar.PersonalisedCompanies(context.Background(), gqlClient, 0, 5, "")
	if err != nil {
		fmt.Printf("Some error %s", err)
		return
	}

	for _, comp := range resp.PersonalisedCompanies {
		fmt.Printf("--- \nName: %s\nSize: %s\nDescription: %s\n", comp.Name, comp.Size.Value, comp.ShortDescription)
	}
}

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
	return base.RoundTrip(clone)
}
