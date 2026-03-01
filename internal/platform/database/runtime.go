package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
	pool *pgxpool.Pool
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

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	rt := &Runtime{pool: pool}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rt.Ping(ctx); err != nil {
		rt.Close()
		return nil, err
	}
	return rt, nil
}

func (r *Runtime) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Runtime) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Runtime) Close() {
	r.pool.Close()
}
