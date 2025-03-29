package middleware

import (
	"context"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/logging"
	"net/http"
	"strings"
)

func AuthMiddleware(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logging.Logg.Warn("Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			username, err := auth.ParseToken(tokenString, cfg)
			if err != nil {
				logging.Logg.Warn("Invalid token", "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
