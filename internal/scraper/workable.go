package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const workableSite = "workable"

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
	retryError := siteCompanyRetryError(workableSite, company)
	resp, err := doWorkableRequest(ctx, httpClient, company, search)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result *WorkableCompany
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, retryError
	}

	return result, nil
}

func doWorkableRequest(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*http.Response, error) {
	url := fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", company)
	retryError := siteCompanyRetryError(workableSite, company)

	payload := WorkablePayload{
		Query:      search,
		Department: []string{},
		Location:   []string{},
		Worktype:   []string{},
		Workplace:  []string{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, siteMarshalError(workableSite, company, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, siteRequestError(workableSite, company, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if isRetryable(err, 0) {
			return nil, retryError
		}
		return nil, fmt.Errorf("%s %s: %w", workableSite, company, err)
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, checkSiteStatusError(workableSite, company, resp.StatusCode)
	}

	return resp, nil
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
