package scraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"khadgar/db/sqlc"
)

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

func (s *Service) DiscoverSite(ctx context.Context, httpClient *http.Client, company sqlc.GetUncheckedCompaniesRow) {
	sites := []siteChecker{
		{teamTailorSite, teamTailorHost, checkTeamTailorJobs, teamTailorCompanyLink, updateCompanyTeamTailor},
		{greenhouseSite, greenhouseHost, checkGreenhouseJobs, greenhouseCompanyLink, updateCompanyGreenhouse},
		{leverSite, leverHost, checkLeverJobs, leverCompanyLink, updateCompanyLever},
	}

	queries := sqlc.New(s.DB.Pool())

	for _, site := range sites {

		// Wait for token to free up before continuing
		// s.Logger.Info("waiting for token", "site", site.name, "company", company.Name)
		if err := s.rateLimiter.Wait(ctx, site.host); err != nil {
			s.Logger.Warn("ctx cancelled/done", "err", err)
			return
		}

		// s.Logger.Info("checking started", "company", company.Name)
		err := site.checkFn(ctx, httpClient, company.UrlSafeName)
		// Found site!
		if err == nil {
			s.Logger.Info("found site", "company", company.Name)
			result := updateResult{
				found:       true,
				workingURL:  site.urlFn(company.UrlSafeName),
				siteName:    site.name,
				shouldRetry: false,
			}
			err := site.updater(ctx, queries, company, result)
			if err != nil {
				s.Logger.Warn("update failed", "company", company.Name)
				return
			}
			s.Logger.Info("saved site", "company", company.Name)
			return
		}

		// Set to retry
		if errors.Is(err, ErrShouldRetry) {
			s.Logger.Warn("retry error", "err", err)
			err := site.updater(ctx, queries, company, updateResult{shouldRetry: true})
			if err != nil {
				s.Logger.Warn("update failed", "company", company.Name)
				return
			}
			return
		} else if errors.Is(err, ErrNotFound) { // Carry on to the next if not found
			s.Logger.Warn("specific site not found", "site", site.name, "err", err)
			err := site.updater(ctx, queries, company, updateResult{})
			if err != nil {
				s.Logger.Warn("update failed", "company", company.Name)
				continue
			}
			continue
		}

		s.Logger.Warn("other error occured! saving for retry for now", "site", site.name, "err", err)
		if updateErr := site.updater(ctx, queries, company, updateResult{shouldRetry: true}); updateErr != nil {
			s.Logger.Warn("update failed", "company", company.Name)
			return
		}
		return
	}
}
