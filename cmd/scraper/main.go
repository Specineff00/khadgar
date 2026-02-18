package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gocolly/colly"
)

//go:embed get_companies.graphql
var getCompaniesQuery string

//go:generate go run github.com/Khan/genqlient ../../genqlient.yaml
func main() {
	// TODO:
	// Make structs for information you need
	// Get list of companies url OR try find their api to get the companies list
	// Inspect company name and target that specific item
	// ForEach through the list
	// Pitstop 1: Print what you have
	// Figure out how to deal with pagination and see if pages can be loaded concurrently
	// If concurrency, then learn how to utilise
	// Figure out way to display all: Bubbletea window? output to file?

	// scrapeCompanyName()
	pingGraphQLEndpoint()
}

func pingGraphQLEndpoint() {
	url := "https://api.exp.welcometothejungle.com/graphql"

	// 1. Define the GraphQL Query (The one you found in the Network tab)
	// I'm using a simplified version for testing

	// 2. Define Variables (Start at 0, get 5 companies)
	variables := map[string]any{
		"offset":      0,
		"limit":       5,
		"companyName": "",
	}

	// 3. Create the JSON payload
	payload := map[string]any{
		"query":     getCompaniesQuery,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer resp.Body.Close()

	var result WTTJCompanies
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Decode Error: %s\n", err)
		return
	}

	for _, comp := range result.Data.PersonalisedCompanies {
		fmt.Printf("--- \nName: %s\nSize: %s\nDescription: %s\n", comp.Name, comp.Size.Value, comp.ShortDescription)
	}
}

func scrapeCompanyName() {
	c := colly.NewCollector()

	fmt.Print("Starting")
	c.OnHTML("a[href^='/companies/']", func(e *colly.HTMLElement) {
		fmt.Println("full", e.Text)
		// 1. Target the Description (the <p> tag)
		// We use the 'Starts With' selector to be safe
		description := e.ChildText("p[class^='sc-']")

		fmt.Println("Description:", description)
	})

	c.Visit("https://app.welcometothejungle.com/companies")
}

type company struct {
	name    string
	size    string
	summary string
}

// TODO: How do move this into internal?

type submission struct {
	URL           string `selector:"span.titleline > a[href]" attr:"href"`
	Title         string `selector:"span.titleline > a"`
	Site          string `selector:"span.sitestr"`
	Id            string
	Score         string `selector:"span.score"`
	TotalComments int
	Comments      []*comment
}

type comment struct {
	Author    string `selector:"a.hnuser"`
	Permalink string `selector:".age a[href]" attr:"href"`
	Comment   string `selector:".comment"`
}

func exampleScraperHandler() {
	// Get the post id as input
	var itemID string
	flag.StringVar(&itemID, "id", "", "hackernews post id")
	flag.Parse()

	if itemID == "" {
		println("Hackernews post id is required")
		os.Exit(1)
	}

	s := &submission{}
	comments := make([]*comment, 0)
	s.Comments = comments
	s.TotalComments = 0

	// Instantiate default collector
	c := colly.NewCollector()
	c.OnHTML("html", func(e *colly.HTMLElement) {
		// Unmarshal the submission struct only on the first page
		if s.Id == "" {
			e.Unmarshal(s)
		}
		s.Id = e.Request.URL.Query().Get("id")

		// Loop over the comment list
		e.ForEach(".comment-tree tr.athing", func(i int, commentElement *colly.HTMLElement) {
			c := &comment{}

			commentElement.Unmarshal(c)
			c.Comment = strings.TrimSpace(c.Comment[:len(c.Comment)-5])
			c.Permalink = commentElement.Request.AbsoluteURL(c.Permalink)
			s.Comments = append(s.Comments, c)
			s.TotalComments += 1
		})
	})

	// Handle pagination
	c.OnHTML("a.morelink", func(e *colly.HTMLElement) {
		c.Visit(e.Request.AbsoluteURL(e.Attr("href")))
	})

	// Go to the submission page
	c.Visit("https://news.ycombinator.com/item?id=" + itemID)

	// Dump json to the standard output (terminal)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(s)
}

type WTTJCompanies struct {
	Data struct {
		PersonalisedCompanies []struct {
			JobLocations []string `json:"jobLocations"`
			Name         string   `json:"name"`
			SectorTags   []struct {
				Value string `json:"value"`
			} `json:"sectorTags"`
			ShortDescription string `json:"shortDescription"`
			Size             struct {
				Value string `json:"value"`
			} `json:"size"`
		} `json:"personalisedCompanies"`
	} `json:"data"`
}
