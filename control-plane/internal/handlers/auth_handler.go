package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/middleware"
	"api-gateway/control-plane/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db        *database.DB
	jwtSecret string
}

func NewAuthHandler(db *database.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 6 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email and password (min 6 chars) required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user := models.User{
		ID:           "usr_" + uuid.New().String()[:8],
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "admin",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	query := `INSERT INTO users (id, email, password_hash, role, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = h.db.Pool.Exec(r.Context(), query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user: " + err.Error()})
		return
	}

	token, err := middleware.GenerateToken(&user, h.jwtSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	respondJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	query := `SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1`
	err := h.db.Pool.QueryRow(r.Context(), query, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := middleware.GenerateToken(&user, h.jwtSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	respondJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var user models.User
	query := `SELECT id, email, role, created_at, updated_at FROM users WHERE id = $1`
	err := h.db.Pool.QueryRow(r.Context(), query, claims.UserID).Scan(
		&user.ID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

