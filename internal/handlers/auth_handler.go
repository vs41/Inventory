package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"freshtrack/internal/auth"
	"freshtrack/internal/config"
	"freshtrack/internal/middleware"
)

type AuthHandler struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var (
		userID       string
		passwordHash string
		role         string
		disabled     bool
	)
	err := h.DB.QueryRow(r.Context(),
		`SELECT id, password_hash, role, disabled FROM users WHERE email = $1`, req.Email,
	).Scan(&userID, &passwordHash, &role, &disabled)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if disabled {
		http.Error(w, "account disabled", http.StatusForbidden)
		return
	}

	if !auth.CheckPassword(passwordHash, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(h.Cfg.JWTSecret, userID, role, h.Cfg.JWTExpiryHours)
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token, Role: role, ID: userID, Email: req.Email})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.CtxUserID).(string)
	role, _ := r.Context().Value(middleware.CtxRole).(string)
	var email string
	var disabled bool
	if err := h.DB.QueryRow(r.Context(), `SELECT email, disabled FROM users WHERE id = $1`, userID).Scan(&email, &disabled); err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": userID, "email": email, "role": role, "disabled": disabled})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
