package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/models"
)

type TiersHandler struct {
	db *database.DB
}

func NewTiersHandler(db *database.DB) *TiersHandler {
	return &TiersHandler{db: db}
}

func (h *TiersHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, rate_limit_per_min, rate_limit_per_sec, monthly_quota, created_at, updated_at 
		FROM tiers 
		ORDER BY rate_limit_per_min ASC
	`
	rows, err := h.db.Pool.Query(r.Context(), query)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query tiers"})
		return
	}
	defer rows.Close()

	tiers := make([]models.Tier, 0)
	for rows.Next() {
		var t models.Tier
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.RateLimitPerMin, &t.RateLimitPerSec, &t.MonthlyQuota, &t.CreatedAt, &t.UpdatedAt); err == nil {
			tiers = append(tiers, t)
		}
	}

	respondJSON(w, http.StatusOK, tiers)
}

func (h *TiersHandler) Update(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "missing tier id"})
		return
	}
	tierID := pathParts[2]

	var req models.Tier
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	query := `
		UPDATE tiers 
		SET rate_limit_per_min = $1, rate_limit_per_sec = $2, monthly_quota = $3, description = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, name, description, rate_limit_per_min, rate_limit_per_sec, monthly_quota, created_at, updated_at
	`
	var t models.Tier
	err := h.db.Pool.QueryRow(r.Context(), query,
		req.RateLimitPerMin, req.RateLimitPerSec, req.MonthlyQuota, req.Description, time.Now().UTC(), tierID,
	).Scan(&t.ID, &t.Name, &t.Description, &t.RateLimitPerMin, &t.RateLimitPerSec, &t.MonthlyQuota, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "tier not found"})
		return
	}

	respondJSON(w, http.StatusOK, t)
}

