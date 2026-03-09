package scraper

import (
	"context"
	"fmt"
	"log/slog"

	"khadgar/db/sqlc"
	"khadgar/internal/platform/database"

	"github.com/Khan/genqlient/graphql"
)

type Service struct {
	RetryConfig RetryConfig
	DB          *database.Runtime
	GQClient    graphql.Client
	Logger      *slog.Logger
}

type Company struct {
	Name             string
	ShortDescription string
	Size             string
	URLSafeName      string
}

type JobRow struct {
	id       string
	title    string
	url      string
	location string
}

type JobProvider interface {
	FetchAndUpsert(ctx context.Context, companyID int, company, search string)
}

func NewService(retry RetryConfig, client graphql.Client, logger *slog.Logger) (*Service, error) {
	db, err := database.NewRuntimeFromEnv()
	if err != nil {
		return nil, err
	}
	return &Service{
		RetryConfig: retry,
		DB:          db,
		GQClient:    client,
		Logger:      logger.With("component", "scraper"),
	}, nil
}

	queries := sqlc.New(s.DB.Pool())
	if err != nil {
	}


}
