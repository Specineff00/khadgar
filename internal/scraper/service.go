package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"github.com/Khan/genqlient/graphql"
	"github.com/jackc/pgx/v5/pgtype"
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
	RetryConfig RetryConfig
	DB          *database.Runtime
	GQClient    graphql.Client
	Logger      *slog.Logger
	wg          *sync.WaitGroup
	rateLimiter *TokenBucketLimiter
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
	URLSafeName      string
}

type JobRow struct {
	id       string
	title    string
	url      string
	location string
}

type JobProvider interface {
	FetchAndUpsert(ctx context.Context, companyID int, company, search string)
}

func NewService(retry RetryConfig, client graphql.Client, logger *slog.Logger) (*Service, error) {
	db, err := database.NewRuntimeFromEnv()
	if err != nil {
		return nil, err
	}
	rl := NewTokenBucketLimiter(2, 3)
	rl.setHostLimiter(workableHost, 0.5, 1)
	rl.setHostLimiter(leverHost, 1, 1)
	return &Service{
		RetryConfig: retry,
		DB:          db,
		GQClient:    client,
		Logger:      logger.With("component", "scraper"),
		wg:          &sync.WaitGroup{},
		rateLimiter: rl,
	}, nil
}

func (s *Service) DiscoverSite(ctx context.Context, httpClient *http.Client, company sqlc.GetUncheckedCompaniesRow) {
	sites := []struct {
		name    string
		host    string
		checkFn func(ctx context.Context, httpClient *http.Client, company string) error
		urlFn   func(company string) string
	}{
		{teamTailorSite, teamTailorHost, checkTeamTailorJobs, teamTailorCompanyLink},
		{greenhouseSite, greenhouseHost, checkGreenhouseJobs, greenhouseCompanyLink},
		{leverSite, leverHost, checkLeverJobs, leverCompanyLink},
		{workableSite, workableHost, checkWorkableJobs, workableCompanyLink},
	}

	queries := sqlc.New(s.DB.Pool())

	for _, site := range sites {
		// Wait for token to free up before continuing
		s.Logger.Info("waiting for token", "site", site.name, "company", company.Name)
		if err := s.rateLimiter.Wait(ctx, site.host); err != nil {
			s.Logger.Warn("ctx cancelled/done", "err", err)
			return
		}

		s.Logger.Info("checking started", "company", company.Name)
		err := site.checkFn(ctx, httpClient, company.UrlSafeName)
		// Found site!
		if err == nil {
			s.Logger.Info("found site", "company", company.Name)
			queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
				Name:            company.Name,
				WorkingUrl:      pgtype.Text{String: site.urlFn(company.UrlSafeName), Valid: true},
				SiteName:        pgtype.Text{String: site.name, Valid: true},
				ShouldRetry:     false,
				AllSitesChecked: true,
			})
			s.Logger.Info("saved site", "company", company.Name)
			return
		}

		// Set to retry
		if errors.Is(err, ErrShouldRetry) {
			s.Logger.Warn("retry error", "err", err)
			queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
				Name:            company.Name,
				WorkingUrl:      pgtype.Text{String: site.urlFn(company.UrlSafeName), Valid: true},
				SiteName:        pgtype.Text{String: site.name, Valid: true},
				ShouldRetry:     true,
				AllSitesChecked: false,
			})
			return
		} else if errors.Is(err, ErrNotFound) { // Carry on to the next if not found
			s.Logger.Warn("specific site not found", "site", site.name, "err", err)
			continue
		} else {
			s.Logger.Warn("other error occured! saving for retry for now", "site", site.name, "err", err)
			queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
				Name:            company.Name,
				WorkingUrl:      pgtype.Text{String: site.urlFn(company.UrlSafeName), Valid: true},
				SiteName:        pgtype.Text{String: site.name, Valid: true},
				ShouldRetry:     true,
				AllSitesChecked: false,
			})
			return
		}
	}

	// All sites visited and nothing found
	s.Logger.Warn("no site found for company", "company", company.Name)
	queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
		Name:            company.Name,
		WorkingUrl:      pgtype.Text{Valid: false},
		SiteName:        pgtype.Text{Valid: false},
		ShouldRetry:     false,
		AllSitesChecked: true,
	})
}

func (s *Service) FeedCompaniesChannel(ctx context.Context) (chan sqlc.GetUncheckedCompaniesRow, error) {
	queries := sqlc.New(s.DB.Pool())
	companies, err := queries.GetUncheckedCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve unchecked companies: %w", err)
	}
	totalCompanies = len(companies)

	companyCh := make(chan sqlc.GetUncheckedCompaniesRow, chanBufferSize)

	go func() {
		defer close(companyCh)
		for _, c := range companies {
			companyCh <- c
		}
	}()

	return companyCh, nil
}

func (s *Service) RunDiscoverSiteWorkers(
	ctx context.Context,
	httpClient *http.Client,
	companyCh <-chan sqlc.GetUncheckedCompaniesRow,
) {
	ctx = attachResponseMetaKey(ctx)
	s.wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer s.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					s.Logger.Error("worker panic", "panic", fmt.Sprint(r))
				}
			}()
			for company := range companyCh {
				s.DiscoverSite(ctx, httpClient, company)
				done.Add(1)
				s.Logger.Info(
					"finished checking out sites",
					"company", company.Name,
					"done", done.Load(),
					"total-companies", totalCompanies,
				)
			}
		}()
	}

	s.wg.Wait()
}
