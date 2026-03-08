package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"khadgar"

	"github.com/Khan/genqlient/graphql"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	RetryConfig RetryConfig
	DB          *database.Runtime
	GQClient    graphql.Client
	Logger      *slog.Logger
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
	URLSafeName      string
}

func NewService(retry RetryConfig, client graphql.Client, logger *slog.Logger) (*Service, error) {
	db, err := database.NewRuntimeFromEnv()
	if err != nil {
		return nil, err
	}
	return &Service{
		RetryConfig: retry,
		DB:          db,
		GQClient:    client,
		Logger:      logger.With("component", "scraper"),
	}, nil
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
		maxPages = 500
	)

	all := make([]Company, 0, 2000)
	seen := make(map[string]struct{}) // dedupe key
	ctx = attachResponseMetaKey(ctx)
	startPage := s.getScrapeStartPage(ctx)

	s.Logger.Info("scrape started",
		"page", startPage,
		"limit", limit,
		"max_pages", maxPages,
	)

	for page := startPage; page < maxPages; page++ {

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
		if err != nil { // Error from retryable or other defined errors
			s.saveScrapePosition(page, false)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				s.logFetchFailed(page, len(all), err)
				return all, fmt.Errorf("scrape canceled: %v", err)
			}
			s.logFetchFailed(page, len(all), err)
			return all, fmt.Errorf("fetch companies page=%d failed after retries: %v", page, err)
		}

		// No reponse errors
		if resp == nil {
			s.saveScrapePosition(page, false)
			s.logFetchFailed(page, len(all), err)
			return all, fmt.Errorf("fetch companies page=%d returned nil response", page)
		}

		cCount := len(resp.PersonalisedCompanies)

		// No more data
		if cCount == 0 {
			s.logFetchComplete(page, len(all))
			s.saveScrapePosition(page, true)
			return all, nil
		}

		// Get the data into shape and in the list
		mappedPage := toCompanies(resp.PersonalisedCompanies)
		all = mergeUnique(all, mappedPage, seen)
		s.logPageFetch(page, len(all))
		s.saveScrapePosition(page, false)

		// Last partial page => likely end.
		// So 32 would be considered added and therefore it's time to break out and finish
		if shouldStop(cCount, limit) {
			s.logFetchComplete(page, len(all))
			s.saveScrapePosition(page, true)
			return all, nil
		}
	}

	// Unlikely scenario but covered to return all
	s.logFetchComplete(maxPages, len(all))
	s.saveScrapePosition(maxPages, false)
	return all, nil
}

func (s *Service) saveScrapePosition(page int, completed bool) {
	ctx := context.Background()
	pool := s.DB.Pool()

	tx, err := pool.Begin(ctx)
	if err != nil {
		s.logDBTransactionStartError(err)
		return
	}

	func() {
		defer tx.Rollback(ctx)

		queries := sqlc.New(pool).WithTx(tx)
		arg := sqlc.UpsertWTTJScrapeMetaDataParams{
			NextPage:  int32(page),
			Completed: completed,
		}
		if err := queries.UpsertWTTJScrapeMetaData(ctx, arg); err != nil {
			s.logDBUpsertError(err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			s.logDBCommitError(err)
		}
	}()
}

func (s *Service) InsertCompaniesBatched(companies []Company) {
	start := time.Now()
	const batchSize = 500
	ctx := context.Background()
	pool := s.DB.Pool()

	for i := 0; i < len(companies); i += batchSize {

		tx, err := pool.Begin(ctx)
		if err != nil {
			s.logDBTransactionStartError(err)
			os.Exit(1)
		}

		func() {
			defer tx.Rollback(ctx)

			queries := sqlc.New(pool).WithTx(tx)
			batch := companies[i:min(i+batchSize, len(companies))]

			for _, c := range batch {
				arg := sqlc.InsertCompanyParams{
					Name:             c.Name,
					ShortDescription: c.ShortDescription,
					Size:             c.Size,
					UrlSafeName:      c.URLSafeName,
				}

				err := queries.InsertCompany(ctx, arg)
				if err != nil {
					s.logDBUpsertError(err)
					return
				}
			}

			if err := tx.Commit(ctx); err != nil {
				s.logDBCommitError(err)
			}
		}()
	}
	elapsed := time.Since(start)
	s.Logger.Info("insert companies complete", "duration", elapsed, "duration_sec", elapsed.Seconds())
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

func attachResponseMetaKey(ctx context.Context) context.Context {
	meta := ResponseMeta{}
	return context.WithValue(ctx, responseMetaKey{}, meta)
}

func (s *Service) getScrapeStartPage(ctx context.Context) int {
	queries := sqlc.New(s.DB.Pool())
	row, err := queries.GetWTTJScrapeMetaData(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0
		}
		s.Logger.Error("failed to load checkpoint", "err", err)
		return 0
	}
	if row.Completed {
		s.Logger.Info("scrape previously completed. restarting")
		return 0
	}
	return int(row.NextPage)
}
