package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const greenhouseSite = "greenhouse"

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

func doGreenhouseRequest(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) (*http.Response, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", company)
	return doRequest(ctx, httpClient, http.MethodGet, url, nil, greenhouseSite, company)
}

func checkGreenhouseJobs(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) error {
	resp, err := doGreenhouseRequest(ctx, httpClient, company)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func FetchGreenhouseJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*GreenhouseCompany, error) {
	resp, err := doGreenhouseRequest(ctx, httpClient, company)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gc *GreenhouseCompany
	if err := json.NewDecoder(resp.Body).Decode(&gc); err != nil {
		return nil, siteCompanyRetryError(greenhouseSite, company)
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

func greenhouseCompanyLink(company string) string {
	return fmt.Sprintf("https://boards.greenhouse.io/%s/", company)
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
