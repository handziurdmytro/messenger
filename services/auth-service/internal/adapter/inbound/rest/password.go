package rest

import (
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/httpjson"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Changes the password of the authenticated user and revokes all refresh tokens
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Param        request body changePasswordRequest true "Current and new password"
// @Success      204
// @Failure      400 {object} errorResponse "Invalid request data or new password rejected"
// @Failure      401 {object} errorResponse "Missing or invalid token"
// @Failure      403 {object} errorResponse "Current password is incorrect"
// @Failure      404 {object} errorResponse "User not found"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /password [patch]
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request, userID auth.UserID) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req changePasswordRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("change password: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
		return
	}

	err := h.changePassword.Execute(r.Context(), usecase.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrPasswordInvalid):
			log.Info("change password: current password is incorrect", logger.KeyUserID, userID)
			writeError(w, http.StatusForbidden, "current password is incorrect")
		case errors.Is(err, auth.ErrPasswordTooShort),
			errors.Is(err, auth.ErrPasswordTooLong),
			errors.Is(err, auth.ErrPasswordUnchanged):
			log.Info("change password: new password rejected", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, auth.ErrUserNotFound):
			log.Info("change password: user not found", logger.KeyUserID, userID)
			writeError(w, http.StatusNotFound, "user not found")
		default:
			log.Error("change password: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("change password: succeeded", logger.KeyUserID, userID)
	w.WriteHeader(http.StatusNoContent)
}
