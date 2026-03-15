package scraper

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrShouldRetry = errors.New("failed request but can be retried")
	ErrNotFound    = errors.New("company not found")
)

func checkSiteStatusError(site, company string, statusCode int) error {
	if isRetryableStatus(statusCode) {
		fmt.Printf("status code %d\n", statusCode)
		return siteCompanyRetryError(site, company, nil)
	} else if statusCode == http.StatusNotFound {
		return fmt.Errorf("%s %s: %w", site, company, ErrNotFound)
	}
	return fmt.Errorf("%s %s: api returned %d", site, company, statusCode)
}

func siteMarshalError(site, company string, err error) error {
	return fmt.Errorf("marshal %s %s payload: %w", site, company, err)
}

func siteRequestError(site, company string, err error) error {
	return fmt.Errorf("request %s %s url: %w", site, company, err)
}

func (s *Service) logDBTransactionStartError(err error) {
	s.Logger.Error("failed to start transaction", "err", err)
}

func (s *Service) logDBUpsertError(err error) {
	s.Logger.Error("insert failed", "err", err)
}

func (s *Service) logDBCommitError(err error) {
	s.Logger.Error("failed to commit", "err", err)
}
