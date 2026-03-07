package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
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
	client := scraper.NewGraphQLClient(url)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service, err := scraper.NewService(retryConfig, client, logger)
	if err != nil {
		logger.Error("db failed to initialise", "err", err)
		os.Exit(1)
	}

	choice := getChoice(service.Logger)

	switch choice {
	case 1:
		scrapeToFile(service)
	case 2:
		scrapeToDB(service)
	case 3:
		insertCompaniesFromFileToDB(service)
	case 4:
		testTeamTailor(service)
	default:
		logger.Error("Not a valid choice!")
		os.Exit(1)
	}
}

func scrapeForCompanies(service *scraper.Service) []scraper.Company {
	companies, err := service.FetchCompanies(context.Background())
	if len(companies) == 0 && err != nil {
		service.Logger.Error("scrape failed with 0 copanies", "err", err)
		os.Exit(1)
	}
	return companies
}

func scrapeToFile(service *scraper.Service) {
	companies := scrapeForCompanies(service)

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
	companies := scrapeForCompanies(service)

	service.InsertCompaniesBatched(companies)
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

	service.InsertCompaniesBatched(companies)
}

func getChoice(logger *slog.Logger) int {
	// 1) CLI argument support (for debugger / automation)
	if len(os.Args) > 1 {
		choice, err := strconv.Atoi(os.Args[1])
		if err != nil || !choiceRangeCondition(choice) {
			logger.Error("invalid CLI choice; use 1 - 4", "arg", os.Args[1], "err", err)
			os.Exit(1)
		}
		return choice
	}

	// 2) Interactive fallback (normal local runs)
	fmt.Println("1. Scrape companies from WTTJ to file")
	fmt.Println("2. Scrape companies from WTTJ to DB")
	fmt.Println("3. Insert companies from file to DB")
	fmt.Println("4. Test workable scrape on starling bank")
	fmt.Print("Choice (1 - 4): ")

	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		logger.Error("failed to read choice from stdin", "err", err)
		os.Exit(1)
	}
	if !choiceRangeCondition(choice) {
		logger.Error("not a valid choice", "choice", choice)
		os.Exit(1)
	}
	return choice
}

func choiceRangeCondition(choice int) bool {
	return choice >= 1 || choice <= 4
}

func testWorkableScrape(service *scraper.Service) {
	httpClient := scraper.NewRESTClient()
	jobs, err := scraper.FetchWorkableJobs(
		context.Background(),
		httpClient,
		"starling-bank",
		"ios",
	)
	if err != nil {
		service.Logger.Error(err.Error())
		os.Exit(1)
	}

	service.Logger.Info("fetch succeeded: %v", "jobs", jobs.Jobs)
	os.Exit(0)
}

func testGreenhouseScrape(service *scraper.Service) {
	httpClient := scraper.NewRESTClient()
	jobs, err := scraper.FetchGreenhouseJobs(
		context.Background(),
		httpClient,
		"monzo",
		"ios",
	)
	if err != nil {
		service.Logger.Error(err.Error())
		os.Exit(1)
	}

	service.Logger.Info("fetch succeeded: %w", "jobs", jobs.Jobs)
	os.Exit(0)
}

func testLeverScrape(service *scraper.Service) {
	httpClient := scraper.NewRESTClient()
	jobs, err := scraper.FetchLeverJobs(
		context.Background(),
		httpClient,
		"moonpig",
		"engineer",
	)
	if err != nil {
		service.Logger.Error(err.Error())
		os.Exit(1)
	}

	service.Logger.Info("fetch succeeded: %w", "jobs", jobs.Jobs)
	os.Exit(0)
}

func testTeamTailor(service *scraper.Service) {
	httpClient := scraper.NewRESTClient()
	jobs, err := scraper.FetchTeamTailorJobs(
		context.Background(),
		httpClient,
		"chip",
		"developer",
	)
	if err != nil {
		service.Logger.Error(err.Error())
		os.Exit(1)
	}

	service.Logger.Info("fetch succeeded: %w", "jobs", jobs.Jobs)
	os.Exit(0)
}
