package database

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var initMigrationSQL string

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 15 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Println("[INFO] Successfully connected to PostgreSQL")

	db := &DB{Pool: pool}
	if err := db.migrate(ctx); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return db, nil
}

func (db *DB) migrate(ctx context.Context) error {
	log.Println("[INFO] Running database migrations...")
	_, err := db.Pool.Exec(ctx, initMigrationSQL)
	if err != nil {
		return fmt.Errorf("failed to execute init migration: %w", err)
	}
	log.Println("[INFO] Database migrations applied successfully")
	return nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

