package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gocolly/colly"
)

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
