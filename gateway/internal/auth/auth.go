package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	KeyPrefix           = "gw_live_"
	KeyCachePrefix      = "key:"
	KeyInvalidationChan = "channel:key_invalidation"
)

var (
	ErrMissingKey = errors.New("missing API key")
	ErrInvalidKey = errors.New("invalid or revoked API key")
	ErrKeyExpired = errors.New("API key has expired")
	ErrForbidden  = errors.New("insufficient scopes for endpoint")
)

type KeyMetadata struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TierID          string     `json:"tier_id"`
	Scopes          []string   `json:"scopes"`
	Status          string     `json:"status"`
	RateLimitPerMin int        `json:"rate_limit_per_min"`
	RateLimitPerSec int        `json:"rate_limit_per_sec"`
	MonthlyQuota    int        `json:"monthly_quota"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CachedAt        time.Time  `json:"-"`
}

type LocalCacheItem struct {
	Metadata  KeyMetadata
	ExpiresAt time.Time
}

type Authenticator struct {
	redisClient *redis.Client
	dbPool      *pgxpool.Pool
	l1Cache     sync.Map
	l1TTL       time.Duration
}

func NewAuthenticator(redisClient *redis.Client, dbPool *pgxpool.Pool, l1TTL time.Duration) *Authenticator {
	auth := &Authenticator{
		redisClient: redisClient,
		dbPool:      dbPool,
		l1TTL:       l1TTL,
	}

	if redisClient != nil {
		go auth.listenInvalidations()
	}

	return auth
}

// ExtractRawKey extracts the raw API key from Authorization header or X-Api-Key
func ExtractRawKey(r *http.Request) (string, error) {
	// 1. Authorization: Bearer <key>
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1]), nil
		}
	}

	// 2. X-Api-Key: <key>
	if apiKey := r.Header.Get("X-Api-Key"); apiKey != "" {
		return strings.TrimSpace(apiKey), nil
	}

	// 3. Query param: ?api_key=<key>
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return strings.TrimSpace(apiKey), nil
	}

	return "", ErrMissingKey
}

// HashKey computes SHA-256 hex string
func HashKey(rawKey string) string {
	h := sha256.New()
	h.Write([]byte(rawKey))
	return hex.EncodeToString(h.Sum(nil))
}

// Authenticate validates the key against L1 (Memory) -> L2 (Redis) -> L3 (PostgreSQL)
func (a *Authenticator) Authenticate(ctx context.Context, rawKey string) (*KeyMetadata, error) {
	if !strings.HasPrefix(rawKey, KeyPrefix) {
		return nil, ErrInvalidKey
	}

	keyHash := HashKey(rawKey)

	// Step 1: Check L1 Local Cache
	if val, ok := a.l1Cache.Load(keyHash); ok {
		item := val.(LocalCacheItem)
		if time.Now().Before(item.ExpiresAt) {
			if item.Metadata.Status != "active" {
				return nil, ErrInvalidKey
			}
			if item.Metadata.ExpiresAt != nil && time.Now().After(*item.Metadata.ExpiresAt) {
				return nil, ErrKeyExpired
			}
			return &item.Metadata, nil
		}
		a.l1Cache.Delete(keyHash)
	}

	// Step 2: Check L2 Redis Cache
	if a.redisClient != nil {
		redisKey := KeyCachePrefix + keyHash
		data, err := a.redisClient.Get(ctx, redisKey).Bytes()
		if err == nil && len(data) > 0 {
			var meta KeyMetadata
			if err := json.Unmarshal(data, &meta); err == nil {
				a.storeL1(keyHash, meta)
				if meta.Status != "active" {
					return nil, ErrInvalidKey
				}
				if meta.ExpiresAt != nil && time.Now().After(*meta.ExpiresAt) {
					return nil, ErrKeyExpired
				}
				return &meta, nil
			}
		}
	}

	// Step 3: Check L3 Database
	if a.dbPool != nil {
		meta, err := a.queryDB(ctx, keyHash)
		if err != nil {
			return nil, ErrInvalidKey
		}

		// Warm L2 Redis
		if a.redisClient != nil {
			payloadBytes, _ := json.Marshal(meta)
			_ = a.redisClient.Set(ctx, KeyCachePrefix+keyHash, payloadBytes, 15*time.Minute).Err()
		}

		// Warm L1 Memory
		a.storeL1(keyHash, *meta)

		if meta.Status != "active" {
			return nil, ErrInvalidKey
		}
		if meta.ExpiresAt != nil && time.Now().After(*meta.ExpiresAt) {
			return nil, ErrKeyExpired
		}
		return meta, nil
	}

	return nil, ErrInvalidKey
}

func (a *Authenticator) queryDB(ctx context.Context, keyHash string) (*KeyMetadata, error) {
	query := `
		SELECT k.id, k.user_id, k.tier_id, k.scopes, k.status, k.expires_at,
		       t.rate_limit_per_min, t.rate_limit_per_sec, t.monthly_quota
		FROM api_keys k
		JOIN tiers t ON k.tier_id = t.id
		WHERE k.key_hash = $1
	`
	var meta KeyMetadata
	var scopesJSON []byte

	err := a.dbPool.QueryRow(ctx, query, keyHash).Scan(
		&meta.ID, &meta.UserID, &meta.TierID, &scopesJSON, &meta.Status, &meta.ExpiresAt,
		&meta.RateLimitPerMin, &meta.RateLimitPerSec, &meta.MonthlyQuota,
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(scopesJSON, &meta.Scopes)
	return &meta, nil
}

func (a *Authenticator) storeL1(keyHash string, meta KeyMetadata) {
	a.l1Cache.Store(keyHash, LocalCacheItem{
		Metadata:  meta,
		ExpiresAt: time.Now().Add(a.l1TTL),
	})
}

// listenInvalidations subscribes to Redis pubsub and purges L1 cache upon key revocation
func (a *Authenticator) listenInvalidations() {
	pubsub := a.redisClient.Subscribe(context.Background(), KeyInvalidationChan)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		keyHash := msg.Payload
		a.l1Cache.Delete(keyHash)
		log.Printf("[INFO] L1 cache invalidated for key hash: %s", keyHash)
	}
}

// ValidateScope checks if key has permission for the method and path
func ValidateScope(scopes []string, method, requestPath string) bool {
	if len(scopes) == 0 {
		return true
	}
	cleanPath := path.Clean(requestPath)

	for _, s := range scopes {
		if s == "*" {
			return true
		}

		// Check "METHOD /path" pattern, e.g. "GET /api/v1/users"
		parts := strings.SplitN(s, " ", 2)
		if len(parts) == 2 {
			scopeMethod := strings.ToUpper(parts[0])
			scopePattern := parts[1]

			if scopeMethod != "*" && scopeMethod != method {
				continue
			}
			if matchPath(scopePattern, cleanPath) {
				return true
			}
		} else {
			// Path pattern only, e.g. "/api/v1/*"
			if matchPath(s, cleanPath) {
				return true
			}
		}
	}
	return false
}

func matchPath(pattern, target string) bool {
	if pattern == "*" || pattern == target {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(target, prefix)
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(target, prefix)
	}
	return false
}

