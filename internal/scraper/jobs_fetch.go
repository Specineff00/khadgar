package scraper

import (
	"context"
	"fmt"
	"khadgar/db/sqlc"
	"net/http"
)

func (s *Service) GetAllJobs(search string) {
	httpClient := NewRESTClient()
	jobCheckers := []jobChecker{
		{teamTailorSite, s.tryTeamTailorAndUpsert},
		{greenhouseSite, s.tryGreenhouseAndUpsert},
		{leverSite, s.tryLeverAndUpsert},
	}

	for _, jobChecker := range jobCheckers {
		ctx := attachResponseMetaKey(context.Background())

		// Load up the channel with companies associated with site for workers
		companyCh, err := s.FeedCompaniesFromSite(ctx, jobChecker.site)
		if err != nil {
			s.Logger.Error("companies failed to load from db", "err", err)
			return
		}
		// Goroutine to fetch and upsert
		s.RunFetchJobsWorkers(ctx, httpClient, search, jobChecker.fetchAndUpsertFn, companyCh)
	}
}

func (s *Service) RunFetchJobsWorkers(
	ctx context.Context,
	httpClient *http.Client,
	search string,
	fetchAndUpsertFn func(
		context.Context,
		*http.Client,
		int,
		string,
		string,
	),
	companyCh <-chan sqlc.GetAllDiscoveredSitesBySiteNameRow,
) {
	ctx = attachResponseMetaKey(ctx)
	s.wg.Add(numWorkers)

	for range numWorkers {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.Logger.Error("worker panic", "panic", fmt.Sprint(r))
			}
		}()
		for company := range companyCh {
			fetchAndUpsertFn(ctx, httpClient, int(company.ID), company.UrlSafeName, search)
			done.Add(1)
			s.Logger.Info(
				"finished searching jobs",
				"company", company,
			)
		}
	}
}

func (s *Service) FetchJobs() {
}
