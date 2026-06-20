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

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(bearer, "Bearer "); ok {
		return after
	}
	return ""
}

func identityFromContext(ctx context.Context) *auth.Identity {
	v, _ := ctx.Value(identityKey).(*auth.Identity)
	return v
}
