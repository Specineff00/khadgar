package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type GreenhouseCompany struct {
	Jobs []struct {
		AbsoluteURL string `json:"absolute_url"`
		Location    struct {
			Name string `json:"name"`
		} `json:"location"`
		ID             int    `json:"id"`
		UpdatedAt      string `json:"updated_at"`
		Title          string `json:"title"`
		FirstPublished string `json:"first_published"`
		Content        string `json:"content"`
	} `json:"jobs"`
}

func FetchGreenhouseJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*GreenhouseCompany, error) {
	site := "greenhouse"
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", company)
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

	var gc *GreenhouseCompany
	if err := json.NewDecoder(resp.Body).Decode(&gc); err != nil {
		return nil, retryErr
	}

	filtered := gc.Jobs[:0]
	for _, job := range gc.Jobs {
		title := strings.ToLower(job.Title)
		if strings.Contains(title, search) {
			filtered = append(filtered, job)
		}
	}
	gc.Jobs = filtered

	return gc, nil
}

func (g GreenhouseCompany) mapToJobRows() []JobRow {
	jobRows := make([]JobRow, 0, len(g.Jobs))

	for _, job := range g.Jobs {
		jobRow := JobRow{
			id:       fmt.Sprintf("%d", job.ID),
			title:    job.Title,
			url:      job.AbsoluteURL,
			location: job.Location.Name,
		}
		jobRows = append(jobRows, jobRow)
	}
	return jobRows
}

func (s *Service) tryGreenhouseAndUpsert(ctx context.Context, companyID int, company, search string) {
	httpClient := NewRESTClient()
	greenhouseCompany, err := FetchGreenhouseJobs(ctx, httpClient, company, search)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	mapped := greenhouseCompany.mapToJobRows()
	s.upsertJobs(ctx, mapped, companyID, search)
}
