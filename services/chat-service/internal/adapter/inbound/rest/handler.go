package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type CreateChatUseCase interface {
	Execute(ctx context.Context, u1, u2 chat.UserID) (chat.Chat, error)
}

type GetChatUseCase interface {
	Execute(ctx context.Context, chatID chat.ID, userID chat.UserID) (chat.Chat, error)
}

type ListChatsUseCase interface {
	Execute(ctx context.Context, userID chat.UserID, limit int, before *usecase.ChatCursor) (usecase.ChatList, error)
}

type Services struct {
	CreateChat CreateChatUseCase
	GetChat    GetChatUseCase
	ListChats  ListChatsUseCase
}

type Handler struct {
	createChat CreateChatUseCase
	getChat    GetChatUseCase
	listChats  ListChatsUseCase
}

func NewHandler(services Services) *Handler {
	return &Handler{
		createChat: services.CreateChat,
		getChat:    services.GetChat,
		listChats:  services.ListChats,
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

	writeJSON(w, toChatResponse(c))
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

	writeJSON(w, toChatResponse(c))
}

func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, before, err := parseChatPage(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	list, err := h.listChats.Execute(r.Context(), userID, limit, before)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	if list.Next != nil {
		w.Header().Set(headerNextCursor, encodeCursor(*list.Next))
	}

	result := make([]chatResponse, 0, len(list.Chats))
	for _, c := range list.Chats {
		result = append(result, toChatResponse(c))
	}
	writeJSON(w, result)
}

type participantResponse struct {
	UserID chat.UserID `json:"user_id"`
}

func (h *Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
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

	participants := c.Participants()
	result := make([]participantResponse, 0, len(participants))
	for _, p := range participants {
		result = append(result, participantResponse{UserID: p})
	}
	writeJSON(w, result)
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

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}
