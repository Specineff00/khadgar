package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	teamTailorSite = "teamtailor"
	teamTailorHost = "teamtailor.com"
)

type TeamTailorCompany struct {
	Jobs []TeamTailorJob
}

type TeamTailorJob struct {
	ID       string
	Title    string
	URL      string
	Location string
	WorkType string // Hybrid/Remote
}

func doTeamTailorRequest(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*http.Response, error) {
	url := fmt.Sprintf("https://%s.%s/jobs?query=%s", company, teamTailorHost, search)
	return doRequest(ctx, httpClient, http.MethodGet, url, nil, teamTailorSite, company)
}

func checkTeamTailorJobs(
	ctx context.Context,
	httpClient *http.Client,
	company string,
) error {
	resp, err := doTeamTailorRequest(ctx, httpClient, company, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func FetchTeamTailorJobs(
	ctx context.Context,
	httpClient *http.Client,
	company, search string,
) (*TeamTailorCompany, error) {
	resp, err := doTeamTailorRequest(ctx, httpClient, company, search)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, siteCompanyRetryError(teamTailorSite, company, err)
	}

	// TeamTailor career pages are server-side rendered HTML (not JSON).
	// We parse the DOM using CSS selectors via goquery.
	//
	// The job listings live inside a <ul id="jobs_list_container">, where
	// each <li> is one job card with this structure:
	//
	//   <li class="block-grid-item ...">
	//     <a href="https://company.teamtailor.com/jobs/12345-job-title">
	//       ...
	//       <span class="text-block-base-link company-link-style" title="Job Title">
	//         Job Title
	//       </span>
	//
	//       <div class="mt-1 text-md">
	//         <span>Department</span>              ← no class attr
	//         <span class="mx-[2px]">·</span>      ← separator (has class)
	//         <span>Location</span>                ← no class attr
	//         <span class="mx-[2px]">·</span>      ← separator (has class)
	//         <span class="inline-flex ...">        ← work type (unique class)
	//           Hybrid
	//         </span>
	//       </div>
	//     </a>
	//   </li>
	//
	// Selector reference:
	//   "#id"            → element with that ID (unique per page)
	//   "tag"            → elements with that tag name
	//   ".class"         → elements that have that CSS class
	//   "[attr]"         → elements that have that attribute (any value)
	//   "parent child"   → child elements nested inside parent (any depth)
	//   ":not([class])"  → elements that do NOT have a class attribute
	//   ".Eq(n)"         → nth match (0-based) from the result set

	result := &TeamTailorCompany{}

	// "#jobs_list_container li" → find all <li> inside the <ul> with that ID.
	// Each <li> is one job card.
	doc.Find("#jobs_list_container li").Each(func(i int, li *goquery.Selection) {
		// Each <li> has a single <a> wrapping the entire card.
		// The href contains the full job URL including the job ID.
		link := li.Find("a")
		href, _ := link.Attr("href")

		// "span[title]" → finds the <span> that has a title attribute.
		// We read from the attribute rather than .Text() because the attribute
		// value is clean, while the text content has extra whitespace/newlines.
		title := li.Find("span[title]").AttrOr("title", "")

		// The metadata div (department, location, work type) is inside "div.mt-1".
		// All three values are in <span> children, separated by dot characters.
		metaDiv := li.Find("div.mt-1")

		// "span.inline-flex" → the work type span is the only one with the
		// "inline-flex" class. Its text is "Hybrid", "Fully Remote", etc.
		// TrimSpace removes whitespace caused by the nested <i> icon element.
		workType := strings.TrimSpace(metaDiv.Find("span.inline-flex").Text())

		// "span:not([class])" → spans WITHOUT a class attribute.
		// This filters out the dot separators (class="mx-[2px]") and the
		// work type (class="inline-flex ..."), leaving only:
		//   Eq(0) = Department
		//   Eq(1) = Location
		location := strings.TrimSpace(metaDiv.Find("span:not([class])").Eq(1).Text())

		result.Jobs = append(result.Jobs, TeamTailorJob{
			ID:       href,
			Title:    title,
			URL:      href,
			Location: location,
			WorkType: workType,
		})
	})

	return result, nil
}

func teamTailorCompanyLink(company string) string {
	return fmt.Sprintf("https://%s.teamtailor.com/#jobs", company)
}

func (t TeamTailorCompany) mapToJobRows() []JobRow {
	jobRows := make([]JobRow, 0, len(t.Jobs))

	for _, job := range t.Jobs {
		jobRow := JobRow{
			id:       job.ID,
			title:    job.Title,
			url:      job.URL,
			location: job.Location,
		}

		jobRows = append(jobRows, jobRow)
	}
	return jobRows
}

func (s *Service) tryTeamTailorAndUpsert(ctx context.Context, companyID int, company, search string) {
	httpClient := NewRESTClient()
	teamTailorCompany, err := FetchTeamTailorJobs(ctx, httpClient, company, search)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}

	mapped := teamTailorCompany.mapToJobRows()
	s.upsertJobs(ctx, mapped, companyID, search)
}
