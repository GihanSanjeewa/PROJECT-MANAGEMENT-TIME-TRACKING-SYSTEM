package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"project-management-system/pkg/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID := GetRequestID(r.Context())
				stack := string(debug.Stack())

				slog.Error("CRITICAL: Panic recovered in HTTP handler",
					"request_id", reqID,
					"panic", rec,
					"stack", stack,
				)

				response.Error(w, fmt.Errorf("a fatal server error was recovered"), reqID)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
