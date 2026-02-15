package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"khadgar/internal/server"
)

func (app *application) serve() error {
	srv := server.New(app.port, app.logger)

	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.Info("completing background tasks", "Addr", srv.Addr)

		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.Info("starting server", "addr", srv.Addr)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", srv.Addr)

	return nil
}

func getPort() (int, error) {
	portFromEnv := 8080

	// Get port from .env file (envdotgo)
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("Invalid Port %w", err)
		}
		portFromEnv = p
	}

	// Flags for override
	port := flag.Int("port", portFromEnv, "api server port")
	flag.Parse()

	// Make sure port is valid
	if *port < 1 || *port > 65535 {
		return 0, fmt.Errorf("port out of range %d", *port)
	}

	return *port, nil
}
