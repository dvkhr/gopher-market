package middleware

import (
	"bytes"
	"context"
	"errors"
	"gopher-market/internal/auth"
	"gopher-market/internal/logging"
	"io"
	"net/http"
	"strings"
	"time"

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

// responseWriterWrapper — это обертка для ResponseWriter, которая записывает HTTP-статус
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает вызов WriteHeader для записи статуса
func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// LoggingMiddleware логирование HTTP-запросов
func LoggingMiddleware(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Извлечение username из контекста
			username, ok := r.Context().Value(UserContextKey).(string)
			if !ok {
				username = "unknown" // Если username отсутствует, используем "unknown"
			}

			// Чтение тела запроса
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				// Восстанавливаем тело запроса, чтобы оно могло быть использовано дальше
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			maskedBody := logging.MaskSensitiveData(string(bodyBytes))

			logger.Info("incoming request",
				"username", username,
				"method", r.Method,
				"url", r.URL.String(),
				"remote_addr", r.RemoteAddr,
				"body", maskedBody, // Логируем тело запроса
			)

			rww := &responseWriterWrapper{ResponseWriter: w}

			next.ServeHTTP(rww, r)

			duration := time.Since(start)
			logger.Info("request completed",
				"username", username,
				"method", r.Method,
				"url", r.URL.String(),
				"status_code", rww.statusCode,
				"duration", duration.String(),
			)
		})
	}
}
