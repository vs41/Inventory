package middleware

import (
	"context"
	"net/http"
	"strings"

	"freshtrack/internal/auth"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			tokenStr := ""
			if strings.HasPrefix(h, "Bearer ") {
				tokenStr = strings.TrimPrefix(h, "Bearer ")
			}
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("access_token")
			}
			if tokenStr == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			claims, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, _ := r.Context().Value(CtxRole).(string)
			if got != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}
