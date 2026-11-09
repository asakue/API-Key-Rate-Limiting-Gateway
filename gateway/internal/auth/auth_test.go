package auth

import (
	"net/http/httptest"
	"testing"
)

func TestExtractRawKey(t *testing.T) {
	// 1. Authorization: Bearer
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer gw_live_1234567890abcdef")
	key1, err := ExtractRawKey(req1)
	if err != nil || key1 != "gw_live_1234567890abcdef" {
		t.Fatalf("expected gw_live_1234567890abcdef, got %s (err: %v)", key1, err)
	}

	// 2. X-Api-Key
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Api-Key", "gw_live_customheader123")
	key2, err := ExtractRawKey(req2)
	if err != nil || key2 != "gw_live_customheader123" {
		t.Fatalf("expected gw_live_customheader123, got %s (err: %v)", key2, err)
	}

	// 3. Query param
	req3 := httptest.NewRequest("GET", "/test?api_key=gw_live_fromquery", nil)
	key3, err := ExtractRawKey(req3)
	if err != nil || key3 != "gw_live_fromquery" {
		t.Fatalf("expected gw_live_fromquery, got %s (err: %v)", key3, err)
	}

	// 4. Missing
	req4 := httptest.NewRequest("GET", "/test", nil)
	_, err = ExtractRawKey(req4)
	if err != ErrMissingKey {
		t.Fatalf("expected ErrMissingKey, got %v", err)
	}
}

func TestHashKey(t *testing.T) {
	raw := "gw_live_0123456789abcdef"
	hash1 := HashKey(raw)
	hash2 := HashKey(raw)

	if hash1 != hash2 {
		t.Fatalf("hash must be deterministic")
	}
	if len(hash1) != 64 {
		t.Fatalf("sha256 hex must be 64 characters, got %d", len(hash1))
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name        string
		scopes      []string
		method      string
		path        string
		expectAllow bool
	}{
		{
			name:        "Wildcard allows everything",
			scopes:      []string{"*"},
			method:      "POST",
			path:        "/api/v1/anything",
			expectAllow: true,
		},
		{
			name:        "Empty scopes allows by default",
			scopes:      []string{},
			method:      "GET",
			path:        "/anything",
			expectAllow: true,
		},
		{
			name:        "Method and path match",
			scopes:      []string{"GET /api/v1/users"},
			method:      "GET",
			path:        "/api/v1/users",
			expectAllow: true,
		},
		{
			name:        "Method mismatch",
			scopes:      []string{"GET /api/v1/users"},
			method:      "POST",
			path:        "/api/v1/users",
			expectAllow: false,
		},
		{
			name:        "Prefix wildcard match",
			scopes:      []string{"GET /api/v1/*"},
			method:      "GET",
			path:        "/api/v1/orders/123",
			expectAllow: true,
		},
		{
			name:        "Prefix wildcard mismatch",
			scopes:      []string{"GET /api/v1/*"},
			method:      "GET",
			path:        "/api/v2/orders",
			expectAllow: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allowed := ValidateScope(tc.scopes, tc.method, tc.path)
			if allowed != tc.expectAllow {
				t.Fatalf("expected allowed=%v, got %v", tc.expectAllow, allowed)
			}
		})
	}
}
