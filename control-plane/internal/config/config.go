package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string
	CORSOrigin    string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/gateway_db?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "gateway_jwt_secret_dev_key_32bytes_min!"
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "*"
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		RedisAddr:     redisAddr,
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret:     jwtSecret,
		CORSOrigin:    corsOrigin,
	}
}

