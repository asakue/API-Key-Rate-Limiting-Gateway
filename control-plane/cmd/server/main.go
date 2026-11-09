package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/control-plane/internal/config"
	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/handlers"
	"api-gateway/control-plane/internal/middleware"
	"api-gateway/control-plane/internal/services"
)

func main() {
	cfg := config.Load()

	log.Printf("[INFO] Initializing Control Plane API on port %s...", cfg.Port)

	// Connect PostgreSQL
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[FATAL] PostgreSQL connection failed: %v", err)
	}
	defer db.Close()

	// Initialize Services
	cacheService := services.NewCacheService(cfg.RedisAddr, cfg.RedisPassword)
	keyService := services.NewKeyService()

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	keysHandler := handlers.NewKeysHandler(db, keyService, cacheService)
	tiersHandler := handlers.NewTiersHandler(db)
	routesHandler := handlers.NewRoutesHandler(db, cacheService)
	metricsHandler := handlers.NewMetricsHandler(db)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"control-plane"}`))
	})

	// Public Auth endpoints
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Protected routes wrapper
	authMiddleware := middleware.AuthRequired(cfg.JWTSecret)

	mux.Handle("GET /api/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

	// API Keys
	mux.Handle("GET /api/keys", authMiddleware(http.HandlerFunc(keysHandler.List)))
	mux.Handle("POST /api/keys", authMiddleware(http.HandlerFunc(keysHandler.Create)))
	mux.Handle("DELETE /api/keys/", authMiddleware(http.HandlerFunc(keysHandler.Revoke)))

	// Tiers
	mux.HandleFunc("GET /api/tiers", tiersHandler.List)
	mux.Handle("PUT /api/tiers/", authMiddleware(http.HandlerFunc(tiersHandler.Update)))

	// Routes
	mux.HandleFunc("GET /api/routes", routesHandler.List)
	mux.Handle("POST /api/routes", authMiddleware(http.HandlerFunc(routesHandler.Create)))
	mux.Handle("DELETE /api/routes/", authMiddleware(http.HandlerFunc(routesHandler.Delete)))

	// Metrics
	mux.Handle("GET /api/metrics/summary", authMiddleware(http.HandlerFunc(metricsHandler.Summary)))
	mux.Handle("GET /api/metrics/timeseries", authMiddleware(http.HandlerFunc(metricsHandler.TimeSeries)))

	// Apply CORS
	handler := middleware.EnableCORS(cfg.CORSOrigin)(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("[INFO] Control Plane API listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] Shutting down Control Plane API gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("[ERROR] Forced shutdown error: %v", err)
	}
	log.Println("[INFO] Control Plane API exited cleanly")
}

