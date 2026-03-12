package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	leverSite = "lever"
	leverHost = "api.lever.co"
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

func doLeverRequest(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) (*http.Response, error) {
	url := fmt.Sprintf("https://%s/v0/postings/%s?mode=json", leverHost, company)
	return doRequest(ctx, httpClient, http.MethodGet, url, nil, leverSite, company)
}

func checkLeverJobs(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) error {
	resp, err := doLeverRequest(ctx, httpClient, company)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func FetchLeverJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*LeverCompany, error) {
	resp, err := doLeverRequest(ctx, httpClient, company)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lc LeverJobs
	if err := json.NewDecoder(resp.Body).Decode(&lc); err != nil {
		return nil, siteCompanyRetryError(leverSite, company)
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

func leverCompanyLink(company string) string {
	return fmt.Sprintf("https://jobs.lever.co/%s", company)
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

func (s *Service) tryLeverAndUpsert(ctx context.Context, companyID int, company, search string) {
	httpClient := NewRESTClient()
	leverCompany, err := FetchLeverJobs(ctx, httpClient, company, search)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	mapped := leverCompany.mapToJobRows()
	s.upsertJobs(ctx, mapped, companyID, search)
}
