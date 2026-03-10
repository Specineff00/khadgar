package scraper

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"github.com/Khan/genqlient/graphql"
	"github.com/jackc/pgx/v5/pgtype"
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
	}, nil
}

// Goes through all sites and checks for existence
func (s *Service) discoverSite(ctx context.Context, httpClient *http.Client, company sqlc.Company) {
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
		err := site.checkFn(ctx, httpClient, company.UrlSafeName)
		// Found site!
		if err == nil {
			queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
				Name:            company.Name,
				WorkingUrl:      pgtype.Text{String: site.urlFn(company.UrlSafeName), Valid: true},
				SiteName:        pgtype.Text{String: site.name, Valid: true},
				ShouldRetry:     false,
				AllSitesChecked: true,
			})
			return
		}

		// Set to retry
		if errors.Is(err, ErrShouldRetry) {
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
			continue
		}
	}

	// All sites visited and nothing found
	queries.UpdateCompanyJobSite(ctx, sqlc.UpdateCompanyJobSiteParams{
		Name:            company.Name,
		WorkingUrl:      pgtype.Text{Valid: false},
		SiteName:        pgtype.Text{Valid: false},
		ShouldRetry:     false,
		AllSitesChecked: true,
	})
}
