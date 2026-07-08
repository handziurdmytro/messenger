package rest

import (
	"encoding/json"
	"errors"
	"net/http"

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
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	u, err := h.getProfile.Execute(r.Context(), user.UserID(userID))
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(u))
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	var req struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.updateProfile.Execute(r.Context(), user.UserID(userID), user.UpdateInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, user.ErrDisplayNameEmpty):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, toResponse(u))
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	users, err := h.searchUsers.Execute(r.Context(), query, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toResponse(u)
	}

	writeJSON(w, http.StatusOK, resp)
}
