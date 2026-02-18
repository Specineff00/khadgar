package main

import (
	"log/slog"
	"os"
	"sync"

	"khadgar/internal/database"
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
	db     database.Service
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

	app := application{
		port:   port,
		db:     database.New(),
		logger: logger,
	}

	// Start the server
	err = app.serve()
	if err != nil {
		os.Exit(1)
	}
}
