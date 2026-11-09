package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/models"
	"api-gateway/control-plane/internal/services"
	"github.com/google/uuid"
)

type RoutesHandler struct {
	db           *database.DB
	cacheService *services.CacheService
}

func NewRoutesHandler(db *database.DB, cs *services.CacheService) *RoutesHandler {
	return &RoutesHandler{db: db, cacheService: cs}
}

func (h *RoutesHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, path_prefix, upstream_url, strip_prefix, timeout_ms, is_active, created_at, updated_at 
		FROM routes 
		ORDER BY created_at ASC
	`
	rows, err := h.db.Pool.Query(r.Context(), query)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query routes"})
		return
	}
	defer rows.Close()

	routes := make([]models.Route, 0)
	for rows.Next() {
		var route models.Route
		if err := rows.Scan(
			&route.ID, &route.Name, &route.PathPrefix, &route.UpstreamURL,
			&route.StripPrefix, &route.TimeoutMS, &route.IsActive, &route.CreatedAt, &route.UpdatedAt,
		); err == nil {
			routes = append(routes, route)
		}
	}

	respondJSON(w, http.StatusOK, routes)
}

func (h *RoutesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.Route
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.PathPrefix = strings.TrimSpace(req.PathPrefix)
	req.UpstreamURL = strings.TrimSpace(req.UpstreamURL)
	if req.PathPrefix == "" || req.UpstreamURL == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "path_prefix and upstream_url are required"})
		return
	}
	if !strings.HasPrefix(req.PathPrefix, "/") {
		req.PathPrefix = "/" + req.PathPrefix
	}
	if req.TimeoutMS <= 0 {
		req.TimeoutMS = 5000
	}

	routeID := "route_" + uuid.New().String()[:8]
	now := time.Now().UTC()

	query := `
		INSERT INTO routes (id, name, path_prefix, upstream_url, strip_prefix, timeout_ms, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := h.db.Pool.Exec(r.Context(), query,
		routeID, req.Name, req.PathPrefix, req.UpstreamURL, req.StripPrefix, req.TimeoutMS, true, now, now,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create route: " + err.Error()})
		return
	}

	req.ID = routeID
	req.IsActive = true
	req.CreatedAt = now
	req.UpdatedAt = now

	// Broadcast route invalidation to Gateways
	_ = h.cacheService.InvalidateRoutes(r.Context())

	respondJSON(w, http.StatusCreated, req)
}

func (h *RoutesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "missing route id"})
		return
	}
	routeID := pathParts[2]

	query := `DELETE FROM routes WHERE id = $1`
	_, err := h.db.Pool.Exec(r.Context(), query, routeID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete route"})
		return
	}

	_ = h.cacheService.InvalidateRoutes(r.Context())

	respondJSON(w, http.StatusOK, map[string]string{"message": "route deleted", "route_id": routeID})
}

