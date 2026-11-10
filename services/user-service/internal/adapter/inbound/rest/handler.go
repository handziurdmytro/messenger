package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type GetProfileUseCase interface {
	Execute(ctx context.Context, id user.ID) (user.User, error)
}

type GetProfilesUseCase interface {
	Execute(ctx context.Context, ids []user.ID) ([]user.User, error)
}

type UpdateProfileUseCase interface {
	Execute(ctx context.Context, id user.ID, input usecase.UpdateProfileInput) (user.User, error)
}

type SearchUsersUseCase interface {
	Execute(ctx context.Context, query string, limit, offset int) ([]user.User, error)
}

type AddContactUseCase interface {
	Execute(ctx context.Context, userID, contactID user.ID) error
}

type RemoveContactUseCase interface {
	Execute(ctx context.Context, userID, contactID user.ID) error
}

type GetContactsUseCase interface {
	Execute(ctx context.Context, userID user.ID, limit, offset int) ([]user.User, error)
}

type Handler struct {
	getProfile    GetProfileUseCase
	updateProfile UpdateProfileUseCase
	searchUsers   SearchUsersUseCase
	addContact    AddContactUseCase
	getContacts   GetContactsUseCase
	removeContact RemoveContactUseCase
	getProfiles   GetProfilesUseCase
}

type Services struct {
	GetProfile    GetProfileUseCase
	UpdateProfile UpdateProfileUseCase
	SearchUsers   SearchUsersUseCase
	AddContact    AddContactUseCase
	GetContacts   GetContactsUseCase
	RemoveContact RemoveContactUseCase
	GetProfiles   GetProfilesUseCase
}

func NewHandler(services Services) *Handler {
	return &Handler{
		getProfile:    services.GetProfile,
		updateProfile: services.UpdateProfile,
		searchUsers:   services.SearchUsers,
		addContact:    services.AddContact,
		getContacts:   services.GetContacts,
		removeContact: services.RemoveContact,
		getProfiles:   services.GetProfiles,
	}
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Deleted     bool   `json:"deleted"`
}

func toResponse(u user.User) userResponse {
	return userResponse{
		ID:          u.ID().String(),
		Email:       u.Email().String(),
		DisplayName: u.DisplayName(),
		AvatarURL:   u.AvatarURL(),
		Deleted:     u.IsDeleted(),
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
	if id := r.PathValue("id"); id != "" {
		userID = id
	}

	u, err := h.getProfile.Execute(r.Context(), user.ID(userID))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
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

	u, err := h.updateProfile.Execute(r.Context(), user.ID(userID), usecase.UpdateProfileInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			log.Info("update_profile: not found", logger.KeyUserID, userID)
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, user.ErrDeleted):
			log.Info("update_profile: user is deleted", logger.KeyUserID, userID)
			writeError(w, http.StatusConflict, "user is deleted")
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

	limit, offset, err := parsePage(r, defaultSearchLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	users, err := h.searchUsers.Execute(r.Context(), query, limit, offset)
	if err != nil {
		log.Error("search_users: failed", "query", query, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("search_users: succeeded", "query", query, "count", len(users))
	writeJSON(w, http.StatusOK, toResponses(users))
}

type addContactRequest struct {
	ContactID string `json:"contact_id"`
}

func (h *Handler) AddContact(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("add_contact: missing user id")
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	var req addContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("add_contact: invalid request body", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ContactID == "" {
		log.Warn("add_contact: missing contact_id", logger.KeyUserID, userID)
		writeError(w, http.StatusBadRequest, "contact_id is required")
		return
	}

	err := h.addContact.Execute(r.Context(), user.ID(userID), user.ID(req.ContactID))
	if err != nil {
		switch {
		case errors.Is(err, contact.ErrCannotAddSelf):
			log.Info("add_contact: tried to add self", logger.KeyUserID, userID)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, contact.ErrAlreadyExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, user.ErrNotFound):
			log.Info("add_contact: contact not found", logger.KeyUserID, userID, "contact_id", req.ContactID)
			writeError(w, http.StatusNotFound, "contact user not found")
		default:
			log.Error("add_contact: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("add_contact: succeeded", logger.KeyUserID, userID, "contact_id", req.ContactID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) GetContacts(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("get_contacts: missing user id")
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	limit, offset, err := parsePage(r, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	contacts, err := h.getContacts.Execute(r.Context(), user.ID(userID), limit, offset)
	if err != nil {
		log.Error("get_contacts: failed", logger.KeyUserID, userID, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("get_contacts: succeeded", logger.KeyUserID, userID, "count", len(contacts))
	writeJSON(w, http.StatusOK, toResponses(contacts))
}

func (h *Handler) RemoveContact(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("remove_contact: missing user id")
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	contactID := r.PathValue("id")
	err := h.removeContact.Execute(r.Context(), user.ID(userID), user.ID(contactID))
	if err != nil {
		switch {
		case errors.Is(err, contact.ErrContactIDEmpty):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, contact.ErrNotFound):
			log.Info("remove_contact: not found", logger.KeyUserID, userID, "contact_id", contactID)
			writeError(w, http.StatusNotFound, "contact not found")
		default:
			log.Error("remove_contact: failed", logger.KeyUserID, userID, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("remove_contact: succeeded", logger.KeyUserID, userID, "contact_id", contactID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "user-rest")

	var ids []user.ID
	for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if id := strings.TrimSpace(raw); id != "" {
			ids = append(ids, user.ID(id))
		}
	}

	users, err := h.getProfiles.Execute(r.Context(), ids)
	if err != nil {
		if errors.Is(err, usecase.ErrTooManyIDs) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d ids are allowed", usecase.MaxProfilesPerRequest))
			return
		}
		log.Error("get_profiles: failed", logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("get_profiles: succeeded", "requested", len(ids), "count", len(users))
	writeJSON(w, http.StatusOK, toResponses(users))
}

func toResponses(users []user.User) []userResponse {
	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toResponse(u)
	}
	return resp
}
