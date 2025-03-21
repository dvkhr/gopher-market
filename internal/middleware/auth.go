package middleware

import (
	"context"
	"errors"
	"gopher-market/internal/auth"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
)

type contextKey string

const UserContextKey contextKey = "username"

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		username, err := auth.ParseToken(tokenString)
		if err != nil {
			var validationErr *jwt.ValidationError
			if errors.As(err, &validationErr) && validationErr.Errors&jwt.ValidationErrorExpired != 0 {
				http.Error(w, "Token expired", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, username)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
