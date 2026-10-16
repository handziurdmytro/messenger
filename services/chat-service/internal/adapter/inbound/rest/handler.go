package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Handler struct {
	createChat *usecase.CreateChatUseCase
	getChat    *usecase.GetChatUseCase
}

func NewHandler(createChat *usecase.CreateChatUseCase, getChat *usecase.GetChatUseCase) *Handler {
	return &Handler{
		createChat: createChat,
		getChat:    getChat,
	}
}

type createChatRequest struct {
	ContactID string `json:"contact_id"`
}

type chatResponse struct {
	ID           chat.ID       `json:"id"`
	Type         chat.Type     `json:"type"`
	Participants []chat.UserID `json:"participants"`
	CreatedAt    string        `json:"created_at"`
}

func toChatResponse(c chat.Chat) chatResponse {
	return chatResponse{
		ID:           c.ID(),
		Type:         c.Type(),
		Participants: c.Participants(),
		CreatedAt:    c.CreatedAt().Format(time.RFC3339Nano),
	}
}

func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req createChatRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contactID, err := chat.ParseUserID(req.ContactID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid contact_id")
		return
	}

	c, err := h.createChat.Execute(r.Context(), userID, contactID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toChatResponse(c))
}

func (h *Handler) GetChat(w http.ResponseWriter, r *http.Request) {
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

	c, err := h.getChat.Execute(r.Context(), chatID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toChatResponse(c))
}

func authenticatedUser(r *http.Request) (chat.UserID, error) {
	return chat.ParseUserID(r.Header.Get("X-User-ID"))
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrChatNotFound):
		writeError(w, http.StatusNotFound, "chat not found")
	case errors.Is(err, usecase.ErrForbidden):
		writeError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, chat.ErrInvalidParticipants),
		errors.Is(err, chat.ErrInvalidUserID),
		errors.Is(err, chat.ErrInvalidChatID):
		writeError(w, http.StatusBadRequest, "invalid chat request")
	default:
		logger.FromContext(r.Context()).Error("chat request failed", logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
