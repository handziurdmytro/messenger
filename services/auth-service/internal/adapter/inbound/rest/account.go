package rest

import (
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/httpjson"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount godoc
// @Summary      Delete account
// @Description  Deletes the authenticated user after password confirmation and publishes user.deleted
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Param        request body deleteAccountRequest true "Password confirmation"
// @Success      204
// @Failure      400 {object} errorResponse "Invalid request data"
// @Failure      401 {object} errorResponse "Missing or invalid token"
// @Failure      403 {object} errorResponse "Password is incorrect"
// @Failure      404 {object} errorResponse "User not found"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /account [delete]
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request, userID auth.UserID) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req deleteAccountRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("delete account: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
		return
	}

	err := h.deleteAccount.Execute(r.Context(), usecase.DeleteAccountInput{
		UserID:   userID,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrPasswordInvalid):
			log.Info("delete account: password is incorrect", logger.KeyUserID, userID)
			writeError(w, http.StatusForbidden, "password is incorrect")
		case errors.Is(err, auth.ErrUserNotFound):
			log.Info("delete account: user not found", logger.KeyUserID, userID)
			writeError(w, http.StatusNotFound, "user not found")
		default:
			log.Error("delete account: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("delete account: succeeded", logger.KeyUserID, userID)
	w.WriteHeader(http.StatusNoContent)
}
