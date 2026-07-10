package logger

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

func HTTPMiddleware(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			ctx := WithRequestID(r.Context(), base, requestID)

			w.Header().Set(RequestIDHeader, requestID)
			r.Header.Set(RequestIDHeader, requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
