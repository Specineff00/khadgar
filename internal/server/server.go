package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"khadgar/internal/database"
)

type Server struct {
	port   int
	db     database.Service
	logger *slog.Logger
}

func NewServer(port int) *http.Server {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	NewServer := &Server{
		port:   port,
		db:     database.New(),
		logger: logger,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
