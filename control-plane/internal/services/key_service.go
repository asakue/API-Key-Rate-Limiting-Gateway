package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const KeyPrefix = "gw_live_"

type KeyService struct{}

func NewKeyService() *KeyService {
	return &KeyService{}
}

// GenerateAPIKey generates a cryptographically secure random API key,
// its prefix for safe display, and its SHA-256 hash for database storage.
func (s *KeyService) GenerateAPIKey() (rawKey string, prefix string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(bytes)
	rawKey = KeyPrefix + randomHex
	prefix = KeyPrefix + randomHex[:8] + "..."
	hash = HashKey(rawKey)

	return rawKey, prefix, hash, nil
}

// HashKey computes SHA-256 hash of the raw API key
func HashKey(rawKey string) string {
	h := sha256.New()
	h.Write([]byte(rawKey))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateKeyFormat checks if key follows `gw_live_<hex>` pattern
func ValidateKeyFormat(rawKey string) error {
	if !strings.HasPrefix(rawKey, KeyPrefix) {
		return errors.New("invalid key format: must start with gw_live_")
	}
	secret := strings.TrimPrefix(rawKey, KeyPrefix)
	if len(secret) < 32 {
		return errors.New("invalid key format: key is too short")
	}
	return nil
}

