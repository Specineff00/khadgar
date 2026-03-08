package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type LeverCompany struct {
	Jobs LeverJobs
}

type LeverJobs []struct {
	AdditionalPlain string `json:"additionalPlain"`
	Categories      struct {
		Location     string   `json:"location"`
		AllLocations []string `json:"allLocations"`
	} `json:"categories"`
	CreatedAt            int64  `json:"createdAt"`
	DescriptionPlain     string `json:"descriptionPlain"`
	ID                   string `json:"id"`
	Title                string `json:"text"`
	Country              string `json:"country"`
	WorkplaceType        string `json:"workplaceType"`
	OpeningPlain         string `json:"openingPlain"`
	DescriptionBodyPlain string `json:"descriptionBodyPlain"`
	HostedURL            string `json:"hostedUrl"`
}

func FetchLeverJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*LeverCompany, error) {
	site := "lever"
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", company)
	retryErr := siteCompanyRetryError(site, company)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, siteRequestError(site, company, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if isRetryable(err, 0) {
			return nil, retryErr
		}
		return nil, fmt.Errorf("%s %s: %w", site, company, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, checkSiteStatusError(site, company, resp.StatusCode)
	}

	var lc LeverJobs
	if err := json.NewDecoder(resp.Body).Decode(&lc); err != nil {
		return nil, retryErr
	}

	filtered := lc[:0]
	for _, job := range lc {
		title := strings.ToLower(job.Title)
		if strings.Contains(title, search) {
			filtered = append(filtered, job)
		}
	}

	return &LeverCompany{Jobs: filtered}, nil
}

func (l LeverCompany) mapToJobRows() []JobRow {
	jobRows := make([]JobRow, 0, len(l.Jobs))

	for _, job := range l.Jobs {
		jobRow := JobRow{
			id:       job.ID,
			title:    job.Title,
			url:      job.HostedURL,
			location: job.Categories.Location,
		}

		jobRows = append(jobRows, jobRow)
	}
	return jobRows
}
