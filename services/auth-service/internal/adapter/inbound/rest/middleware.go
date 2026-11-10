package rest

import (
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type authenticatedHandler func(http.ResponseWriter, *http.Request, auth.UserID)

func (h *Handler) requireAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}

		identity, err := h.tokenSvc.Authenticate(auth.Token(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		next(w, r, auth.UserID(identity.UserID))
	}
}
