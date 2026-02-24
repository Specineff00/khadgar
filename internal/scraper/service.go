package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"khadgar/internal/platform/database"

	"khadgar"

	"github.com/Khan/genqlient/graphql"
)

type Service struct {
	RetryConfig RetryConfig
	DB          database.Service
	GQClient    graphql.Client
	Logger      *slog.Logger
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
}

func NewService(retry RetryConfig, client graphql.Client, logger *slog.Logger) *Service {
	return &Service{
		RetryConfig: retry,
		DB:          database.New(),
		GQClient:    client,
		Logger:      logger.With("component", "scraper"),
	}
}

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
func (s *Service) FetchCompanies(ctx context.Context) ([]Company, error) {
	const (
		limit    = 50
		maxPages = 200
	)

	all := make([]Company, 0, 2000)
	seen := make(map[string]struct{}) // dedupe key

	s.Logger.Info("scrap started",
		"limit", limit,
		"max_pages", maxPages,
	)

	for page := range maxPages {

		var resp *khadgar.PersonalisedCompaniesResponse
		err := s.doWithRetry(ctx, func(ctx context.Context) (statusCode int, err error) {
			r, err := khadgar.PersonalisedCompanies(
				ctx,
				s.GQClient,
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
				s.logFetchFailed(page, len(all), err)
				return all, fmt.Errorf("scrape canceled: %v", err)
			}
			s.logFetchFailed(page, len(all), err)
			return all, fmt.Errorf("fetch companies page=%d failed after retries: %v", page, err)
		}
		if resp == nil {
			s.logFetchFailed(page, len(all), err)
			return all, fmt.Errorf("fetch companies page=%d returned nil response", page)
		}

		cCount := len(resp.PersonalisedCompanies)
		if cCount == 0 {
			// No more data
			s.logFetchComplete(page, len(all))
			return all, nil
		}

		mappedPage := toCompanies(resp.PersonalisedCompanies)
		all = mergeUnique(all, mappedPage, seen)
		s.logPageFetch(page, len(all))

		// Last partial page => likely end.
		// So 32 would be considered added and therefore it's time to break out and finish
		if shouldStop(cCount, limit) {
			s.logFetchComplete(page, len(all))
			return all, nil
		}
	}

	s.logFetchComplete(999, len(all))
	return all, nil
}

func (s *Service) logFetchComplete(page, fetched int) {
	s.Logger.Info("scrape completed",
		"page", page,
		"fetched", fetched,
	)
}

func (s *Service) logFetchFailed(page, fetched int, err error) {
	s.Logger.Info("scrape failed",
		"page", page,
		"fetched", fetched,
		"error", err.Error(),
	)
}

func (s *Service) logPageFetch(page, fetched int) {
	s.Logger.Info("current page fetch",
		"page", page,
		"fetched", fetched,
	)
}
