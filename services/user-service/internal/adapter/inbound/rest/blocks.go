package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type BlockUserUseCase interface {
	Execute(ctx context.Context, userID, blockedID user.ID) error
}

type UnblockUserUseCase interface {
	Execute(ctx context.Context, userID, blockedID user.ID) error
}

type ListBlockedUseCase interface {
	Execute(ctx context.Context, userID user.ID, limit, offset int) ([]user.User, error)
}

type blockUserRequest struct {
	UserID string `json:"user_id"`
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	var req blockUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("block_user: invalid request body", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.blockUser.Execute(r.Context(), user.ID(userID), user.ID(req.UserID))
	if err != nil {
		switch {
		case errors.Is(err, block.ErrBlockedIDEmpty), errors.Is(err, block.ErrCannotBlockSelf):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, block.ErrAlreadyBlocked):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, user.ErrNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		default:
			log.Error("block_user: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("block_user: succeeded", logger.KeyUserID, userID, "blocked_id", req.UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	blockedID := r.PathValue("id")
	err := h.unblockUser.Execute(r.Context(), user.ID(userID), user.ID(blockedID))
	if err != nil {
		switch {
		case errors.Is(err, block.ErrBlockedIDEmpty):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, block.ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			log.Error("unblock_user: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("unblock_user: succeeded", logger.KeyUserID, userID, "blocked_id", blockedID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	limit, offset, err := parsePage(r, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	users, err := h.listBlocked.Execute(r.Context(), user.ID(userID), limit, offset)
	if err != nil {
		log.Error("list_blocked: failed", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toResponses(users))
}
