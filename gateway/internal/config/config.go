package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	RedisAddr          string
	RedisPassword      string
	DatabaseURL        string
	DefaultUpstreamURL string
	L1CacheTTLSeconds  int
	WorkerPoolSize     int
	BatchFlushIntervalMs int
	BatchMaxSize       int
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/gateway_db?sslmode=disable"
	}

	defaultUpstream := os.Getenv("DEFAULT_UPSTREAM_URL")
	if defaultUpstream == "" {
		defaultUpstream = "http://localhost:8080"
	}

	l1TTL, _ := strconv.Atoi(os.Getenv("L1_CACHE_TTL_SECONDS"))
	if l1TTL <= 0 {
		l1TTL = 60 // 1 minute local in-memory L1 cache
	}

	return &Config{
		Port:                 port,
		RedisAddr:            redisAddr,
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		DatabaseURL:          dbURL,
		DefaultUpstreamURL:   defaultUpstream,
		L1CacheTTLSeconds:    l1TTL,
		WorkerPoolSize:       4,
		BatchFlushIntervalMs: 2000,
		BatchMaxSize:         500,
	}
}

