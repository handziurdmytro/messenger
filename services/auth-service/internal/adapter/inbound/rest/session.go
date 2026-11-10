package rest

import (
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/pkg/httpjson"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

// Logout godoc
// @Summary      Logout
// @Description  Revokes the given refresh token
// @Tags         auth
// @Accept       json
// @Param        request body refreshRequest true "Refresh token"
// @Success      204
// @Failure      400 {object} errorResponse "Invalid request data"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req refreshRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("logout: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
		return
	}

	if err := h.logout.Execute(r.Context(), auth.RefreshToken(req.RefreshToken)); err != nil {
		if errors.Is(err, auth.ErrRefreshInvalid) {
			writeError(w, http.StatusBadRequest, "refresh token is required")
			return
		}
		log.Error("logout: failed", logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("logout: succeeded")
	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll godoc
// @Summary      Logout from all devices
// @Description  Revokes every refresh token of the authenticated user
// @Tags         auth
// @Security     BearerAuth
// @Success      204
// @Failure      401 {object} errorResponse "Missing or invalid token"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /logout-all [post]
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request, userID auth.UserID) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	if err := h.logoutAll.Execute(r.Context(), userID); err != nil {
		log.Error("logout all: failed", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("logout all: succeeded", logger.KeyUserID, userID)
	w.WriteHeader(http.StatusNoContent)
}
