package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ssback/internal/services"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		claims, err := services.ParseToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil || !claims.IsAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetClaims(r *http.Request) *services.Claims {
	c, _ := r.Context().Value(ClaimsKey).(*services.Claims)
	return c
}

func extractToken(r *http.Request) string {
	// 1. Authorization header (mobile)
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// 2. Cookie (web)
	if c, err := r.Cookie("access_token"); err == nil {
		val := c.Value
		if strings.HasPrefix(val, "Bearer ") {
			val = strings.TrimPrefix(val, "Bearer ")
		}
		return val
	}
	return ""
}
