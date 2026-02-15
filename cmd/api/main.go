package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"khadgar/internal/server"
)

// NOTES:
// - DB and app are two separate processes
// - .env file has variables we use for development and therefore should not be committed
// - godotenv is a library that gives go functionality to read and parse .env files
// - _ "github.com/joho/godotenv/autoload" specifically loads the .env in to the environments on init
// This init happens from server package and as you can see the server package is imported here, which causes
// godotenv to init before getPort and subsequently Getenv gets called.
// This .env file can also be read directly from docker-compose files if they are in the same folder

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	port, err := getPort()
	if err != nil {
		log.Fatal(err)
	}

	server := server.New(port)

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
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

	if *port < 1 || *port > 65535 {
		return 0, fmt.Errorf("port out of range %d", *port)
	}

	return *port, nil
}
