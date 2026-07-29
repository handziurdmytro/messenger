package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type Handler struct {
	getProfile    *usecase.GetProfileUseCase
	updateProfile *usecase.UpdateProfileUseCase
	searchUsers   *usecase.SearchUsersUseCase
}

func NewHandler(
	getProfile *usecase.GetProfileUseCase,
	updateProfile *usecase.UpdateProfileUseCase,
	searchUsers *usecase.SearchUsersUseCase,
) *Handler {
	return &Handler{
		getProfile:    getProfile,
		updateProfile: updateProfile,
		searchUsers:   searchUsers,
	}
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func toResponse(u user.User) userResponse {
	return userResponse{
		ID:          string(u.ID),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("get_profile: missing user id")
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	u, err := h.getProfile.Execute(r.Context(), user.ID(userID))
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			log.Info("get_profile: not found", logger.KeyUserID, userID)
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("get_profile: failed", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("get_profile: succeeded", logger.KeyUserID, userID)
	writeJSON(w, http.StatusOK, toResponse(u))
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("update_profile: missing user id")
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	var req struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("update_profile: invalid request body", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.updateProfile.Execute(r.Context(), user.ID(userID), user.UpdateInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserNotFound):
			log.Info("update_profile: not found", logger.KeyUserID, userID)
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, user.ErrDisplayNameEmpty):
			log.Info("update_profile: display name empty", logger.KeyUserID, userID)
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("update_profile: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("update_profile: succeeded", logger.KeyUserID, userID)
	writeJSON(w, http.StatusOK, toResponse(u))
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	query := r.URL.Query().Get("q")

	users, err := h.searchUsers.Execute(r.Context(), query, 20)
	if err != nil {
		log.Error("search_users: failed", "query", query, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("search_users: succeeded", "query", query, "count", len(users))

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toResponse(u)
	}

	writeJSON(w, http.StatusOK, resp)
}
