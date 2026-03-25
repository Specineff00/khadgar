package scraper

import (
	"context"
	"fmt"
	"net/http"

	"khadgar/db/sqlc"
)

func (s *Service) GetAllJobs(search string) {
	httpClient := NewRESTClient()
	jobCheckers := []jobChecker{
		{teamTailorSite, s.tryTeamTailorAndUpsert},
		{greenhouseSite, s.tryGreenhouseAndUpsert},
		{leverSite, s.tryLeverAndUpsert},
	}

	s.Logger.Info("starting job fetch", "search", search)
	for _, jobChecker := range jobCheckers {
		ctx := attachResponseMetaKey(context.Background())

		// Load up the channel with companies associated with site for workers
		companyCh, err := s.FeedCompaniesFromSite(ctx, jobChecker.site)
		count := len(companyCh)
		s.Logger.Info("companies loaded", "site", jobChecker.site, "count", count)
		if err != nil {
			s.Logger.Error("companies failed to load from db", "err", err)
			return
		}
		// Goroutine to fetch and upsert
		s.RunFetchJobsWorkers(
			ctx,
			httpClient,
			search,
			jobChecker.site,
			jobChecker.fetchAndUpsertFn,
			companyCh,
		)
	}
}

func (s *Service) RunFetchJobsWorkers(
	ctx context.Context,
	httpClient *http.Client,
	search, site string,
	fetchAndUpsertFn func(
		context.Context,
		*http.Client,
		int,
		string,
		string,
	),
	companyCh <-chan sqlc.GetAllDiscoveredSitesBySiteNameRow,
) {
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
				fetchAndUpsertFn(ctx, httpClient, int(company.ID), company.UrlSafeName, search)
				done.Add(1)
				s.Logger.Info(
					"finished searching jobs",
					"company", company.UrlSafeName,
					"site", site,
				)
			}
		}()
	}
	s.wg.Wait()
}
