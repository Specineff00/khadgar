package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	workableSite = "workable"
	workableHost = "apply.workable.com"
)

type WorkableCompany struct {
	Total int `json:"total"`
	Jobs  []struct {
		ID        int    `json:"id"`
		ShortCode string `json:"shortcode"`
		Title     string `json:"title"`
		Remote    bool   `json:"remote"`
		Location  struct {
			Country string `json:"country"`
			City    string `json:"city"`
		} `json:"location"`
		Locations []struct {
			Country string `json:"country"`
			City    string `json:"city"`
		} `json:"locations"`
		Published time.Time `json:"published"`
	} `json:"results"`
	NextPage string `json:"nextPage"`
}

type WorkablePayload struct {
	Query      string   `json:"query"`
	Department []string `json:"department"`
	Location   []string `json:"location"`
	Workplace  []string `json:"workplace"`
	Worktype   []string `json:"worktype"`
}

func doWorkableRequest(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*http.Response, error) {
	url := fmt.Sprintf("https://%s/api/v3/accounts/%s/jobs", workableHost, company)

	payload := WorkablePayload{
		Query:      search,
		Department: []string{},
		Location:   []string{},
		Worktype:   []string{},
		Workplace:  []string{},
	}

	return doRequest(ctx, httpClient, http.MethodPost, url, payload, workableSite, company)
}

func checkWorkableJobs(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) error {
	resp, err := doWorkableRequest(ctx, httpClient, company, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func fetchWorkableJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*WorkableCompany, error) {
	resp, err := doWorkableRequest(ctx, httpClient, company, search)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result *WorkableCompany
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, siteCompanyRetryError(workableSite, company, err)
	}

	return result, nil
}

func workableCompanyLink(company string) string {
	return fmt.Sprintf("https://apply.workable.com/%s", company)
}

func workableJobLink(company string, id int) string {
	return fmt.Sprintf("https://apply.workable.com/%s/jobs/%d", company, id)
}

func (w WorkableCompany) mapToJobRows(company string) []JobRow {
	jobRows := make([]JobRow, 0, len(w.Jobs))

	for _, job := range w.Jobs {
		jobRow := JobRow{
			id:       fmt.Sprintf("%d", job.ID),
			title:    job.Title,
			url:      workableJobLink(company, job.ID),
			location: job.Location.City,
		}
		jobRows = append(jobRows, jobRow)
	}
	return jobRows
}

func (s *Service) tryWorkableAndUpsert(ctx context.Context, companyID int, company, search string) {
	httpClient := NewRESTClient()
	workableCompany, err := fetchWorkableJobs(ctx, httpClient, company, search)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	mapped := workableCompany.mapToJobRows(company)
	s.Logger.Info("attempting to insert jobs", "jobs", mapped)
	s.upsertJobs(ctx, mapped, companyID, search)
}
