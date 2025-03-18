package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var Logg *slog.Logger

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
		start := time.Now()

		crw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(crw, r)

		duration := time.Since(start)

		Logg.Info(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", fmt.Sprintf("%v: %v", crw.status, http.StatusText(crw.status)),
			slog.Duration("duration", duration),
			"size", crw.size,
		)
	})
}
