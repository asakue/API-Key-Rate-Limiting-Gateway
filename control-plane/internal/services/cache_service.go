package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"api-gateway/control-plane/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	KeyCachePrefix        = "key:"
	KeyInvalidationChan   = "channel:key_invalidation"
	RouteInvalidationChan = "channel:route_invalidation"
	DefaultKeyTTL         = 15 * time.Minute
)

type CacheService struct {
	client *redis.Client
}

func NewCacheService(redisAddr, password string) *CacheService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[WARN] Redis ping failed (cache will retry on demand): %v", err)
	} else {
		log.Printf("[INFO] Connected to Redis at %s", redisAddr)
	}

	return &CacheService{client: rdb}
}

// CacheKey stores the API key metadata in Redis
func (c *CacheService) CacheKey(ctx context.Context, keyHash string, payload models.KeyCachePayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal key cache payload: %w", err)
	}

	redisKey := KeyCachePrefix + keyHash
	if err := c.client.Set(ctx, redisKey, data, DefaultKeyTTL).Err(); err != nil {
		return fmt.Errorf("failed to set key in redis: %w", err)
	}
	return nil
}

// InvalidateKey removes the key from Redis and notifies all Gateway instances via Pub/Sub
func (c *CacheService) InvalidateKey(ctx context.Context, keyHash string) error {
	redisKey := KeyCachePrefix + keyHash
	_ = c.client.Del(ctx, redisKey).Err()

	// Broadcast invalidation event
	if err := c.client.Publish(ctx, KeyInvalidationChan, keyHash).Err(); err != nil {
		return fmt.Errorf("failed to publish key invalidation: %w", err)
	}
	return nil
}

// InvalidateRoutes signals all Gateway instances to reload their route table
func (c *CacheService) InvalidateRoutes(ctx context.Context) error {
	if err := c.client.Publish(ctx, RouteInvalidationChan, "reload").Err(); err != nil {
		return fmt.Errorf("failed to publish route invalidation: %w", err)
	}
	return nil
}

