package scraper

import (
	"context"
	"os"
	"time"

	"khadgar/db/sqlc"
)

func (s *Service) InsertCompaniesBatched(companies []Company) {
	start := time.Now()
	const batchSize = 500
	ctx := context.Background()
	pool := s.DB.Pool()

	for i := 0; i < len(companies); i += batchSize {

		tx, err := pool.Begin(ctx)
		if err != nil {
			s.logDBTransactionStartError(err)
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
					UrlSafeName:      c.URLSafeName,
				}

				err := queries.InsertCompany(ctx, arg)
				if err != nil {
					s.logDBUpsertError(err)
					return
				}
			}

			if err := tx.Commit(ctx); err != nil {
				s.logDBCommitError(err)
			}
		}()
	}
	elapsed := time.Since(start)
	s.Logger.Info("insert companies complete", "duration", elapsed, "duration_sec", elapsed.Seconds())
}

func (s *Service) upsertJobs(
	ctx context.Context,
	jobs []JobRow,
	companyID int,
	search string,
) {
	pool := s.DB.Pool()

	tx, err := pool.Begin(ctx)
	if err != nil {
		s.logDBTransactionStartError(err)
		return
	}

	func() {
		defer tx.Rollback(ctx)

		queries := sqlc.New(pool).WithTx(tx)

		for _, job := range jobs {

			arg := sqlc.UpsertJobParams{
				CompanyID:  int64(companyID),
				ExternalID: job.id,
				SearchTerm: search,
				Title:      job.title,
				Url:        job.url,
				Location:   job.location,
			}

			if err := queries.UpsertJob(ctx, arg); err != nil {
				s.logDBUpsertError(err)
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			s.logDBCommitError(err)
		}
	}()
}
