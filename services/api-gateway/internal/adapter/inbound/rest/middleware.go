package rest

import (
	"context"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
)

type contextKey string

const identityKey contextKey = "identity"

func authMiddleware(authenticator *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			identity, err := authenticator.Authenticate(auth.Token(token))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), identityKey, &identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(bearer, "Bearer "); ok {
		return after
	}
	return r.URL.Query().Get("token")
}

func identityFromContext(ctx context.Context) *auth.Identity {
	v, _ := ctx.Value(identityKey).(*auth.Identity)
	return v
}
