package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/middleware"
	"api-gateway/control-plane/internal/models"
	"api-gateway/control-plane/internal/services"
	"github.com/google/uuid"
)

type KeysHandler struct {
	db           *database.DB
	keyService   *services.KeyService
	cacheService *services.CacheService
}

func NewKeysHandler(db *database.DB, ks *services.KeyService, cs *services.CacheService) *KeysHandler {
	return &KeysHandler{
		db:           db,
		keyService:   ks,
		cacheService: cs,
	}
}

func (h *KeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req models.CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "Default Key"
	}
	if req.TierID == "" {
		req.TierID = "tier_free"
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"*"}
	}

	// Verify tier exists and get limits
	var tier models.Tier
	tierQuery := `SELECT id, name, rate_limit_per_min, rate_limit_per_sec, monthly_quota FROM tiers WHERE id = $1`
	err := h.db.Pool.QueryRow(r.Context(), tierQuery, req.TierID).Scan(
		&tier.ID, &tier.Name, &tier.RateLimitPerMin, &tier.RateLimitPerSec, &tier.MonthlyQuota,
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "specified tier does not exist"})
		return
	}

	rawKey, prefix, hash, err := h.keyService.GenerateAPIKey()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate key"})
		return
	}

	keyID := "key_" + uuid.New().String()[:12]
	scopesJSON, _ := json.Marshal(req.Scopes)
	now := time.Now().UTC()

	insertQuery := `
		INSERT INTO api_keys (id, user_id, tier_id, name, key_prefix, key_hash, scopes, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = h.db.Pool.Exec(r.Context(), insertQuery,
		keyID, claims.UserID, tier.ID, req.Name, prefix, hash, scopesJSON, "active", req.ExpiresAt, now,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save key in database: " + err.Error()})
		return
	}

	// Warm cache in Redis
	cachePayload := models.KeyCachePayload{
		ID:              keyID,
		UserID:          claims.UserID,
		TierID:          tier.ID,
		Scopes:          req.Scopes,
		Status:          "active",
		RateLimitPerMin: tier.RateLimitPerMin,
		RateLimitPerSec: tier.RateLimitPerSec,
		MonthlyQuota:    tier.MonthlyQuota,
		ExpiresAt:       req.ExpiresAt,
	}
	_ = h.cacheService.CacheKey(r.Context(), hash, cachePayload)

	apiKey := models.APIKey{
		ID:              keyID,
		UserID:          claims.UserID,
		TierID:          tier.ID,
		TierName:        tier.Name,
		Name:            req.Name,
		KeyPrefix:       prefix,
		Scopes:          req.Scopes,
		Status:          "active",
		RateLimitPerMin: tier.RateLimitPerMin,
		RateLimitPerSec: tier.RateLimitPerSec,
		MonthlyQuota:    tier.MonthlyQuota,
		ExpiresAt:       req.ExpiresAt,
		CreatedAt:       now,
	}

	respondJSON(w, http.StatusCreated, models.CreateKeyResponse{
		APIKey: apiKey,
		RawKey: rawKey,
	})
}

func (h *KeysHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	query := `
		SELECT k.id, k.user_id, k.tier_id, t.name as tier_name, k.name, k.key_prefix, 
		       k.scopes, k.status, k.expires_at, k.created_at, k.revoked_at,
		       t.rate_limit_per_min, t.rate_limit_per_sec, t.monthly_quota
		FROM api_keys k
		JOIN tiers t ON k.tier_id = t.id
		WHERE k.user_id = $1
		ORDER BY k.created_at DESC
	`
	rows, err := h.db.Pool.Query(r.Context(), query, claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query keys"})
		return
	}
	defer rows.Close()

	keys := make([]models.APIKey, 0)
	for rows.Next() {
		var k models.APIKey
		var scopesJSON []byte
		err := rows.Scan(
			&k.ID, &k.UserID, &k.TierID, &k.TierName, &k.Name, &k.KeyPrefix,
			&scopesJSON, &k.Status, &k.ExpiresAt, &k.CreatedAt, &k.RevokedAt,
			&k.RateLimitPerMin, &k.RateLimitPerSec, &k.MonthlyQuota,
		)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(scopesJSON, &k.Scopes)
		keys = append(keys, k)
	}

	respondJSON(w, http.StatusOK, keys)
}

func (h *KeysHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Extract key ID from URL path (e.g. /api/keys/{id})
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "missing key id"})
		return
	}
	keyID := pathParts[2]

	var keyHash string
	selectQuery := `SELECT key_hash FROM api_keys WHERE id = $1 AND user_id = $2`
	err := h.db.Pool.QueryRow(r.Context(), selectQuery, keyID, claims.UserID).Scan(&keyHash)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}

	now := time.Now().UTC()
	updateQuery := `UPDATE api_keys SET status = 'revoked', revoked_at = $1 WHERE id = $2`
	_, err = h.db.Pool.Exec(r.Context(), updateQuery, now, keyID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke key"})
		return
	}

	// Invalidate in Redis and notify Gateways
	_ = h.cacheService.InvalidateKey(r.Context(), keyHash)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "key revoked successfully",
		"key_id":  keyID,
	})
}

