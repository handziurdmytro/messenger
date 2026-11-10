package rest

import (
	"net/http"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
)

type createGroupRequest struct {
	Title     string        `json:"title"`
	MemberIDs []chat.UserID `json:"member_ids"`
}

type renameGroupRequest struct {
	Title string `json:"title"`
}

type addParticipantsRequest struct {
	UserIDs []chat.UserID `json:"user_ids"`
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createGroupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.createGroup.Execute(r.Context(), usecase.CreateGroupInput{
		Owner:   userID,
		Title:   req.Title,
		Members: req.MemberIDs,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, toChatResponse(c))
}

func (h *Handler) RenameGroup(w http.ResponseWriter, r *http.Request) {
	var req renameGroupRequest
	h.changeGroup(w, r, &req, func() usecase.GroupChange { return usecase.Rename(req.Title) })
}

func (h *Handler) AddParticipants(w http.ResponseWriter, r *http.Request) {
	var req addParticipantsRequest
	h.changeGroup(w, r, &req, func() usecase.GroupChange { return usecase.AddParticipants(req.UserIDs) })
}

func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	target, err := chat.ParseUserID(r.PathValue("user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if target == userID {
		h.DeleteChat(w, r)
		return
	}

	h.changeGroup(w, r, nil, func() usecase.GroupChange { return usecase.RemoveParticipant(target) })
}

func (h *Handler) changeGroup(w http.ResponseWriter, r *http.Request, body any, change func() usecase.GroupChange) {
	userID, err := authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID, err := chat.ParseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chat ID")
		return
	}

	if body != nil {
		if err := decodeJSON(w, r, body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	c, err := h.updateGroup.Execute(r.Context(), chatID, userID, change())
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, toChatResponse(c))
}
