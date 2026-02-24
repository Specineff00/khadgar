package main

import (
	"log/slog"
	"os"
	"sync"

	"khadgar/internal/platform/database"
)

// NOTES:
// - DB and app are two separate processes
// - .env file has variables we use for development and therefore should not be committed
// - godotenv is a library that gives go functionality to read and parse .env files
// - _ "github.com/joho/godotenv/autoload" specifically loads the .env in to the environments on init
// This init happens from server package and as you can see the server package is imported here, which causes
// godotenv to init before getPort and subsequently Getenv gets called.
// This .env file can also be read directly from docker-compose files if they are in the same folder

type application struct {
	port   int
	db     *database.Runtime
	logger *slog.Logger
	wg     sync.WaitGroup
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Port setup
	port, err := getPort()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	dbRuntime, err := database.NewRuntimeFromEnv()
	if err != nil {
		logger.Error("database failed to init", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := dbRuntime.Close(); err != nil {
			logger.Error("database failed to close")
		}
	}()

	app := application{
		port:   port,
		db:     dbRuntime,
		logger: logger,
	}

	// Start the server
	if err = app.serve(); err != nil {
		os.Exit(1)
	}
}
