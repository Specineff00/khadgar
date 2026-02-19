package main

import (
	_ "embed"
	"math/rand"
	"time"

	"khadgar/internal/scraper"
)

//go:generate go run github.com/Khan/genqlient ../../genqlient.yaml
func main() {
	// TODO:
	// Pitstop 1: Print what you have
	// Figure out how to deal with pagination and see if pages can be loaded concurrently
	// If concurrency, then learn how to utilise
	// Figure out way to display all: Bubbletea window? output to file?

	// scrapeCompanyName()
	pingGraphQLEndpoint()
}

func pingGraphQLEndpoint() {
	scraper.FetchCompanies()
}

func waitWithJitter() {
	var base time.Duration = 300
	jitter := time.Duration(rand.Intn(300)-150) * time.Millisecond
	time.Sleep(base + jitter)
}
