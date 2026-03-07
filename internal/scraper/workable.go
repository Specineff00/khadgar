package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

// !! Change to urlSafeName in WTTJ company scraper !!
// This may need x<

// Database
// - Create migration files for adding
// -- website which exists workable, teamTailor, greenhouse, leaver
// -- actual link to concatted website
// -- last visited date

// Research for actual requests
// What is the actual request from each webstie?
//
// Workable: https://apply.workable.com/api/v3/accounts/[company]/jobs
//
// TeamTailor: https://[company].teamtailor.com/jobs/
// -- This needs actual scraping of elements
// -- https://footasylum.teamtailor.com/jobs/show_more?page=2 needed
//
// Greenhouse: https://job-boards.greenhouse.io/company
// This, like TeamTailor may need actual scraping of elements
// https://boards-api.greenhouse.io/v1/boards/[slug]/job-boards
//
// Lever: https://jobs.lever.co/moonpig/
// This, like TeamTailor may need actual scraping of elements
// https://api.lever.co/v0/postings/moonpig?mode=json
//
// With each website
// - Check if Exists using
// -- Exists may show actual site but it may show page with no information so false positive
// -- If exists save basic database info,
// - Find request for certain jobs
// - Is it REST or GQL?
// - Create the single request

func FetchWorkableJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*WorkableCompany, error) {
	site := "workable"
	url := fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", company)
	retryError := siteCompanyRetryError(site, company)

	payload := WorkablePayload{
		Query:      search,
		Department: []string{},
		Location:   []string{},
		Worktype:   []string{},
		Workplace:  []string{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, siteMarshalError(site, company, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, siteRequestError(site, company, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if isRetryable(err, 0) {
			return nil, retryError
		}
		return nil, fmt.Errorf("%s %s: %w", site, company, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, checkSiteStatusError(site, company, resp.StatusCode)
	}

	var result *WorkableCompany
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, retryError
	}

	return result, nil
}

func workableJobLink(company, id string) string {
	return fmt.Sprintf("https://apply.workable.com/%s/jobs/%s", company, id)
}
