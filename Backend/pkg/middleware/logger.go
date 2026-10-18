package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (d *responseWriterDelegator) WriteHeader(code int) {
	d.statusCode = code
	d.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := GetRequestID(r.Context())

		delegator := &responseWriterDelegator{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(delegator, r)

		duration := time.Since(start)

		slog.Info("HTTP Request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", delegator.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
