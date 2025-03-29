package middleware

import (
	"bytes"
	"context"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/logging"
	"io"
	"net/http"
	"time"
)

type contextKey string

const UserContextKey contextKey = "username"

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
			ctx := r.Context()
			select {
			case <-ctx.Done():
				http.Error(w, "Request canceled", http.StatusServiceUnavailable)
				return
			default:
			}

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

func AuthMiddleware(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logging.Logg.Warn("Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Проверяем токен
			username, err := auth.ParseToken(authHeader, cfg)
			if err != nil {
				logging.Logg.Warn("Invalid token", "error", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			} // Добавляем username в контекст запроса
			ctx := context.WithValue(r.Context(), "username", username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
