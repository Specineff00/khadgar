package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"github.com/Khan/genqlient/graphql"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	chanBufferSize = 256
	numWorkers     = 50
)

type Service struct {
	RetryConfig RetryConfig
	DB          *database.Runtime
	GQClient    graphql.Client
	Logger      *slog.Logger
	wg          *sync.WaitGroup
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
	return &Service{
		RetryConfig: retry,
		DB:          db,
		GQClient:    client,
		Logger:      logger.With("component", "scraper"),
		wg:          &sync.WaitGroup{},
	}, nil
}

func (s *Service) DiscoverSite(ctx context.Context, httpClient *http.Client, company sqlc.GetUncheckedCompaniesRow) {
	sites := []struct {
		name    string
		checkFn func(ctx context.Context, httpClient *http.Client, company string) error
		urlFn   func(company string) string
	}{
		{workableSite, checkWorkableJobs, workableCompanyLink},
		{greenhouseSite, checkGreenhouseJobs, greenhouseCompanyLink},
		{leverSite, checkLeverJobs, leverCompanyLink},
		{teamTailorSite, checkTeamTailorJobs, teamTailorCompanyLink},
	}

	queries := sqlc.New(s.DB.Pool())

	for _, site := range sites {
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
		}

		// Carry on to the next if not found
		if errors.Is(err, ErrNotFound) {
			s.Logger.Warn("specific site not found", "site", site.name, "err", err)
			continue
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
	s.wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer s.wg.Done()
			for company := range companyCh {
				s.DiscoverSite(ctx, httpClient, company)
			}
		}()
	}

	s.wg.Wait()
}
