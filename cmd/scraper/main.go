package main

import (
	"context"
	_ "embed"
	"log"
	"log/slog"
	"os"
	"time"

	"khadgar/internal/platform/database"
	"khadgar/internal/scraper"
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

	service := scraper.Service{
		RetryConfig: retryConfig,
		DB:          database.New(),
		GQClient:    client,
		Logger:      logger,
	}

	companies, err := service.FetchCompanies(context.Background())
	if err != nil {
		log.Print(err.Error())
	}

	log.Printf("%v", companies)
}
