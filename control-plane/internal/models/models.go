package models

import (
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Tier struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	RateLimitPerMin int       `json:"rate_limit_per_min"`
	RateLimitPerSec int       `json:"rate_limit_per_sec"`
	MonthlyQuota    int       `json:"monthly_quota"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type APIKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TierID    string     `json:"tier_id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"`
	KeyHash   string     `json:"-"`
	Scopes    []string   `json:"scopes"`
	Status    string     `json:"status"` // "active", "revoked", "expired"
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	// Joined fields
	TierName        string `json:"tier_name,omitempty"`
	RateLimitPerMin int    `json:"rate_limit_per_min,omitempty"`
	RateLimitPerSec int    `json:"rate_limit_per_sec,omitempty"`
	MonthlyQuota    int    `json:"monthly_quota,omitempty"`
}

type Route struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PathPrefix  string    `json:"path_prefix"`
	UpstreamURL string    `json:"upstream_url"`
	StripPrefix bool      `json:"strip_prefix"`
	TimeoutMS   int       `json:"timeout_ms"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateKeyRequest struct {
	Name      string     `json:"name"`
	TierID    string     `json:"tier_id"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateKeyResponse struct {
	APIKey
	RawKey string `json:"raw_key"` // Shown ONCE to user
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type MetricSummary struct {
	TotalRequests    int64   `json:"total_requests"`
	Total2xx         int64   `json:"total_2xx"`
	Total4xx         int64   `json:"total_4xx"`
	Total5xx         int64   `json:"total_5xx"`
	Total429         int64   `json:"total_429"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	ActiveKeysCount  int64   `json:"active_keys_count"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Total     int64     `json:"total"`
	Success   int64     `json:"success"` // 2xx
	ClientErr int64     `json:"client_err"` // 4xx (excluding 429)
	Throttled int64     `json:"throttled"` // 429
	ServerErr int64     `json:"server_err"` // 5xx
}

// KeyCachePayload is stored in Redis under key:<sha256_hash>
type KeyCachePayload struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	TierID          string    `json:"tier_id"`
	Scopes          []string  `json:"scopes"`
	Status          string    `json:"status"`
	RateLimitPerMin int       `json:"rate_limit_per_min"`
	RateLimitPerSec int       `json:"rate_limit_per_sec"`
	MonthlyQuota    int       `json:"monthly_quota"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

