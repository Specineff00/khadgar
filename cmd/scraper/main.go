package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"khadgar/internal/scraper"

	_ "github.com/joho/godotenv/autoload"
)

//go:generate go run github.com/Khan/genqlient ../../genqlient.yaml
func main() {
	retryConfig := scraper.RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   250 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		JitterFrac:  0.2,
	}

	url := "https://api.exp.welcometothejungle.com/graphql"
	client := scraper.NewClient(url)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service, err := scraper.NewService(retryConfig, client, logger)
	if err != nil {
		logger.Error("db failed to initialise", "err", err)
		os.Exit(1)
	}

	fmt.Println("1. Scrape to file")
	fmt.Println("2. Scrape to DB")
	fmt.Print("Choice (1 or 2): ")
	var choice int
	fmt.Scanln(&choice)
	switch choice {
	case 1:
		// scrape to file
		scrapeToFile(service)
	case 2:
		// scrape to DB
		logger.Error("Not yet implemented!")
		os.Exit(1)
	default:
		logger.Error("Not a valid choice!")
		os.Exit(1)
	}
}

func scrapeToFile(service *scraper.Service) {
	companies, err := service.FetchCompanies(context.Background())
	if len(companies) == 0 && err != nil {
		service.Logger.Error("scrape failed", "err", err, "count", len(companies))
		os.Exit(1)
	}

	service.Logger.Info("encoding")
	data, err := json.MarshalIndent(companies, "", "  ")
	if err != nil {
		service.Logger.Error("encode failed", "err", err)
		os.Exit(1)
	}

	service.Logger.Info("encoded!")

	service.Logger.Info("writing to json")
	path := "companies.json"
	if err = os.WriteFile(path, data, 0o644); err != nil {
		service.Logger.Error("write failed", "err", err)
		os.Exit(1)
	}

	service.Logger.Info("wrote companies to file", "path", path, "count", len(companies))
}
