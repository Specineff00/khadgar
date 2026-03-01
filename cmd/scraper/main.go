package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"khadgar/db/sqlc"
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
	fmt.Println("3. File to DB")
	fmt.Print("Choice (1, 2 or 3): ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		scrapeToFile(service)
	case 2:
		scrapeToDB(service)
	case 3:
		insertCompaniesFromFileToDB(service)
	default:
		logger.Error("Not a valid choice!")
		os.Exit(1)
	}
}

func scrape(service *scraper.Service) []scraper.Company {
	companies, err := service.FetchCompanies(context.Background())
	if len(companies) == 0 && err != nil {
		service.Logger.Error("scrape failed with 0 copanies", "err", err)
		os.Exit(1)
	}
	return companies
}

func scrapeToFile(service *scraper.Service) {
	companies := scrape(service)

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

func scrapeToDB(service *scraper.Service) {
	companies := scrape(service)

	insertCompaniesBatched(service, companies)
}

func insertCompaniesFromFileToDB(service *scraper.Service) {
	data, err := os.ReadFile("companies.json")
	if err != nil {
		service.Logger.Error("read failed", "err", err)
		os.Exit(1)
	}

	var companies []scraper.Company

	if err := json.Unmarshal(data, &companies); err != nil {
		service.Logger.Error("decode failed", "err", err)
		os.Exit(1)
	}

	insertCompaniesBatched(service, companies)
}

func insertCompaniesBatched(service *scraper.Service, companies []scraper.Company) {
	start := time.Now()
	const batchSize = 500
	ctx := context.Background()
	pool := service.DB.Pool()

	for i := 0; i < len(companies); i += batchSize {

		tx, err := pool.Begin(ctx)
		if err != nil {
			service.Logger.Error("failed to start transaction", "err", err)
			os.Exit(1)
		}

		func() {
			defer tx.Rollback(ctx)

			queries := sqlc.New(pool).WithTx(tx)
			batch := companies[i:min(i+batchSize, len(companies))]

			for _, c := range batch {
				arg := sqlc.InsertCompanyParams{
					Name:             c.Name,
					ShortDescription: c.ShortDescription,
					Size:             c.Size,
				}

				err := queries.InsertCompany(ctx, arg)
				if err != nil {
					service.Logger.Error("insert failed")
					return
				}
			}

			if err := tx.Commit(ctx); err != nil {
				service.Logger.Error("failed to commit", "err", err)
			}
		}()
	}
	elapsed := time.Since(start)
	service.Logger.Info("insert companies complete", "duration", elapsed, "duration_sec", elapsed.Seconds())
}
