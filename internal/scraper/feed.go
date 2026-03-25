package scraper

import (
	"context"
	"fmt"
	"khadgar/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

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

func (s *Service) FeedCompaniesFromSite(ctx context.Context, site string) (chan sqlc.GetAllDiscoveredSitesBySiteNameRow, error) {
	queries := sqlc.New(s.DB.Pool())
	companies, err := queries.GetAllDiscoveredSitesBySiteName(ctx, pgtype.Text{String: site, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve %s companies", site)
	}

	totalCompanies = len(companies)
	companyCh := make(chan sqlc.GetAllDiscoveredSitesBySiteNameRow, chanBufferSize)
	go func() {
		defer close(companyCh)
		for _, c := range companies {
			companyCh <- c
		}
	}()

	return companyCh, nil
}
