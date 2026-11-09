package storage

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func ConnectRedis(addr, password string) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 20,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[WARN] Gateway Redis ping failed: %v", err)
		return &RedisClient{Client: rdb}, err
	}

	log.Printf("[INFO] Gateway connected to Redis at %s", addr)
	return &RedisClient{Client: rdb}, nil
}

func (r *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return r.Client.Subscribe(ctx, channel)
}

func (r *RedisClient) Close() {
	if r.Client != nil {
		_ = r.Client.Close()
	}
}
