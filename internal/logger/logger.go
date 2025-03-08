package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

var Logg slog.Logger

func init() {
	logg := zap.Must(zap.NewProduction())

	defer logg.Sync()
	logger := slog.New(zapslog.NewHandler(logg.Core(), nil))
	Logg = *logger
}

type (
	responseData struct {
		status int
		size   int
	}
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)
		duration := time.Since(start)

		Logg.Info(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", fmt.Sprintf("%v: %v", responseData.status, http.StatusText(responseData.status)),
			slog.Duration("duration", duration),
			"size", responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}
