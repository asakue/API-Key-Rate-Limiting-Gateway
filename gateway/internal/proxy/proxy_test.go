package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"api-gateway/gateway/internal/auth"
	"api-gateway/gateway/internal/ratelimit"
	"api-gateway/gateway/internal/router"
)

func TestGatewayProxy(t *testing.T) {
	// 1. Setup mock upstream server that inspects incoming headers
	var receivedConsumerID, receivedKeyID, receivedRemaining string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedConsumerID = r.Header.Get("X-Consumer-Id")
		receivedKeyID = r.Header.Get("X-Key-Id")
		receivedRemaining = r.Header.Get("X-RateLimit-Remaining")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"path":   r.URL.Path,
		})
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	// 2. Setup Authenticator with pre-populated L1 cache
	authenticator := auth.NewAuthenticator(nil, nil, 5*time.Minute)

	validRawKey := "gw_live_11223344556677889900aabbccddeeff"
	validHash := auth.HashKey(validRawKey)

	// Inject active key into L1 cache
	authenticator.Authenticate(nil, "gw_live_dummy") // no-op trigger
	// Add directly via reflect or by using Authenticator methods
	// We can add a helper or mock
	testMeta := auth.KeyMetadata{
		ID:              "key_test_123",
		UserID:          "usr_dev_456",
		TierID:          "tier_starter",
		Scopes:          []string{"GET /api/v1/*"},
		Status:          "active",
		RateLimitPerMin: 100,
		RateLimitPerSec: 20,
		MonthlyQuota:    50000,
	}

	// We can instantiate an Authenticator and store in its L1 cache via exported or reflection,
	// or test individual scenarios.
}

