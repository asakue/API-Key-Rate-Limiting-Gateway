package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type EchoResponse struct {
	Message    string              `json:"message"`
	Timestamp  time.Time           `json:"timestamp"`
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	Headers    map[string][]string `json:"headers"`
	Query      map[string][]string `json:"query"`
	Body       string              `json:"body,omitempty"`
	ConsumerID string              `json:"consumer_id,omitempty"`
	KeyID      string              `json:"key_id,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		applyDelay(r)
		w.Header().Set("Content-Type", "application/json")
		users := []map[string]any{
			{"id": "usr_1", "name": "Alice Developer", "tier": "Pro"},
			{"id": "usr_2", "name": "Bob Architect", "tier": "Starter"},
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   users,
			"path":   r.URL.Path,
		})
	})

	mux.HandleFunc("/api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		applyDelay(r)
		w.Header().Set("Content-Type", "application/json")
		orders := []map[string]any{
			{"order_id": "ord_101", "total_usd": 249.99, "status": "shipped"},
			{"order_id": "ord_102", "total_usd": 89.50, "status": "processing"},
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   orders,
			"path":   r.URL.Path,
		})
	})

	// Generic handler for anything else
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		applyDelay(r)

		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		resp := EchoResponse{
			Message:    "Request proxied successfully via API Key & Rate Limiting Gateway",
			Timestamp:  time.Now().UTC(),
			Method:     r.Method,
			Path:       r.URL.Path,
			Headers:    r.Header,
			Query:      r.URL.Query(),
			Body:       string(bodyBytes),
			ConsumerID: r.Header.Get("X-Consumer-Id"),
			KeyID:      r.Header.Get("X-Key-Id"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Target Mock Backend listening on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func applyDelay(r *http.Request) {
	if delayStr := r.URL.Query().Get("delay_ms"); delayStr != "" {
		if d, err := strconv.Atoi(delayStr); err == nil && d > 0 {
			time.Sleep(time.Duration(d) * time.Millisecond)
		}
	} else if delayStr = r.Header.Get("X-Mock-Delay"); delayStr != "" {
		if d, err := strconv.Atoi(delayStr); err == nil && d > 0 {
			time.Sleep(time.Duration(d) * time.Millisecond)
		}
	}
}

