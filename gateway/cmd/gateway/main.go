package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/gateway/internal/auth"
	"api-gateway/gateway/internal/config"
	"api-gateway/gateway/internal/proxy"
	"api-gateway/gateway/internal/ratelimit"
	"api-gateway/gateway/internal/router"
	"api-gateway/gateway/internal/storage"
	"api-gateway/gateway/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	log.Printf("[INFO] Starting API Key & Rate Limiting Gateway on port %s...", cfg.Port)

	// Connect Redis
	redisStorage, err := storage.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Printf("[WARN] Gateway continuing without direct Redis: %v", err)
	}
	defer redisStorage.Close()

	// Connect PostgreSQL
	pgCtx, pgCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pgCancel()

	pgPool, err := pgxpool.New(pgCtx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("[WARN] PostgreSQL connection pool init error: %v", err)
	} else {
		defer pgPool.Close()
	}

	// Initialize Telemetry Collector
	collector := telemetry.NewCollector(
		pgPool,
		10000,
		time.Duration(cfg.BatchFlushIntervalMs)*time.Millisecond,
		cfg.BatchMaxSize,
	)
	defer collector.Close()

	// Initialize Rate Limiter
	limiter := ratelimit.NewLimiter(redisStorage.Client)

	// Initialize Authenticator
	authenticator := auth.NewAuthenticator(
		redisStorage.Client,
		pgPool,
		time.Duration(cfg.L1CacheTTLSeconds)*time.Second,
	)

	// Initialize Dynamic Router
	dynamicRouter, err := router.NewDynamicRouter(pgPool, redisStorage.Client, cfg.DefaultUpstreamURL)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize router: %v", err)
	}

	// Initialize Gateway Reverse Proxy
	gatewayProxy := proxy.NewGatewayProxy(authenticator, limiter, dynamicRouter, collector)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      gatewayProxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("[INFO] Gateway reverse proxy listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Gateway server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] Shutting down Gateway gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[ERROR] Forced shutdown error: %v", err)
	}
	log.Println("[INFO] Gateway exited cleanly")
}

