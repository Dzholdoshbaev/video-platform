package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				// Panic is reserved for broken invariants/unexpected failures, never business errors.
				logger.Error("recovered HTTP panic",
					"request_id", RequestIDFromContext(r.Context()), "panic", recovered,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "internal_server_error", "message": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
