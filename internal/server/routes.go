package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/health", healthHandler)

	r.Route("/v1/scraper", func(r chi.Router) {
		r.Post("/jobs/{jobType}", triggerScrapeHandler)
	})
	return corsMiddleware(r)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	//	resp, err := json.Marshal(s.db.Health())
	//
	// if err != nil {
	// http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)
	//
	//			return
	//	}
	//
	// w.Header().Set("Content-Type", "application/json")
	//
	//	if _, err := w.Write(resp); err != nil {
	//			log.Printf("Failed to write response: %v", err)
	//		}
}

func triggerScrapeHandler(w http.ResponseWriter, r *http.Request) {
	jobType := chi.URLParam(r, "jobType")
	jobType = strings.ToLower(jobType)

	log.Printf("This is the post of scraper %s", jobType)
}
