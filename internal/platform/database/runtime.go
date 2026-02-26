package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

type Runtime struct {
	db *sql.DB
}

func NewRuntimeFromEnv() (*Runtime, error) {
	database := os.Getenv("DB_DATABASE")
	password := os.Getenv("DB_PASSWORD")
	username := os.Getenv("DB_USERNAME")
	port := os.Getenv("DB_PORT")
	host := os.Getenv("DB_HOST")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username, password, host, port, database,
	)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	rt := &Runtime{db: db}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rt.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return rt, nil
}

func (r *Runtime) DB() *sql.DB {
	return r.db
}

func (r *Runtime) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Runtime) Close() error {
	return r.db.Close()
}
