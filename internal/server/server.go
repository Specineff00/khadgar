package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func New(port int, logger *slog.Logger) *http.Server {
	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      NewRouter(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	return server
}
