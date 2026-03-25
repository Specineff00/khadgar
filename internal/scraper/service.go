package scraper

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"github.com/Khan/genqlient/graphql"
)

const (
	chanBufferSize = 256
	numWorkers     = 5
)

var (
	done           atomic.Int64
	totalCompanies int
)

type Service struct {
	RetryConfig    RetryConfig
	DB             *database.Runtime
	GQClient       graphql.Client
	Logger         *slog.Logger
	wg             *sync.WaitGroup
	rateLimiter    *TokenBucketLimiter
	siteErrorCount map[string]atomic.Int32
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
	URLSafeName      string
}

type updateResult struct {
	found       bool
	workingURL  string
	siteName    string
	shouldRetry bool
}

type siteChecker struct {
	name    string
	host    string
	checkFn func(ctx context.Context, httpClient *http.Client, company string) error
	urlFn   func(company string) string
	updater func(
		ctx context.Context,
		queries *sqlc.Queries,
		company sqlc.GetUncheckedCompaniesRow,
		result updateResult,
	) error
}

type jobChecker struct {
	site             string
	fetchAndUpsertFn func(
		context.Context,
		*http.Client,
		int,
		string,
		string,
	)
}

type JobRow struct {
	id       string
	title    string
	url      string
	location string
}

func NewService(retry RetryConfig, client graphql.Client, logger *slog.Logger) (*Service, error) {
	db, err := database.NewRuntimeFromEnv()
	if err != nil {
		return nil, err
	}
	rl := NewTokenBucketLimiter(2, 3)
	rl.setHostLimiter(leverHost, 1, 1)
	return &Service{
		RetryConfig:    retry,
		DB:             db,
		GQClient:       client,
		Logger:         logger.With("component", "scraper"),
		wg:             &sync.WaitGroup{},
		rateLimiter:    rl,
		siteErrorCount: make(map[string]atomic.Int32),
	}, nil
}
