package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

var Logg *slog.Logger

type contextKey string

const UserContextKey contextKey = "username"

func init() {
	logger := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Logg = slog.New(logger)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)

}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//start := time.Now()

		username, ok := r.Context().Value(UserContextKey).(string)
		if !ok {
			username = "User is not authorized"
		}
		/*var logMessage string

		if ok && username != "" {
			logMessage = "User: " + username
		} else {
			logMessage = "User is not authorized"
		}*/

		crw := &loggingResponseWriter{ResponseWriter: w}
		ctx := context.WithValue(r.Context(), UserContextKey, username)

		//duration := time.Since(start)

		Logg.Info(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", fmt.Sprintf("%v: %v", crw.status, http.StatusText(crw.status)),
			//slog.Duration("duration", duration),
			//"size", crw.size,
			slog.String("User: ", username),
		)
		next.ServeHTTP(crw, r.WithContext(ctx))
	})
}
