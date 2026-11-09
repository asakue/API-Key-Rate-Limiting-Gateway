package services

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	ks := NewKeyService()
	rawKey, prefix, hash, err := ks.GenerateAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(rawKey, "gw_live_") {
		t.Fatalf("expected key to start with gw_live_, got %s", rawKey)
	}

	if len(rawKey) != 8+64 { // "gw_live_" (8) + 32 bytes hex (64) = 72
		t.Fatalf("expected key length 72, got %d", len(rawKey))
	}

	if !strings.HasPrefix(prefix, "gw_live_") || !strings.HasSuffix(prefix, "...") {
		t.Fatalf("invalid prefix preview format: %s", prefix)
	}

	if len(hash) != 64 {
		t.Fatalf("expected sha256 hash length 64, got %d", len(hash))
	}

	// Verify HashKey function matches
	if HashKey(rawKey) != hash {
		t.Fatalf("hash from HashKey does not match generated hash")
	}

	// Verify format validation
	if err := ValidateKeyFormat(rawKey); err != nil {
		t.Fatalf("expected key to pass validation: %v", err)
	}

	if err := ValidateKeyFormat("invalid_prefix_12345"); err == nil {
		t.Fatalf("expected invalid prefix to fail validation")
	}
}

