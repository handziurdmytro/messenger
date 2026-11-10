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

type DeleteChatUseCase interface {
	Execute(ctx context.Context, chatID chat.ID, userID chat.UserID) error
}

type CreateGroupUseCase interface {
	Execute(ctx context.Context, in usecase.CreateGroupInput) (chat.Chat, error)
}

type UpdateGroupUseCase interface {
	Execute(ctx context.Context, chatID chat.ID, by chat.UserID, change usecase.GroupChange) (chat.Chat, error)
}

type Services struct {
	CreateChat  CreateChatUseCase
	GetChat     GetChatUseCase
	ListChats   ListChatsUseCase
	DeleteChat  DeleteChatUseCase
	CreateGroup CreateGroupUseCase
	UpdateGroup UpdateGroupUseCase
}

type Handler struct {
	createChat  CreateChatUseCase
	getChat     GetChatUseCase
	listChats   ListChatsUseCase
	deleteChat  DeleteChatUseCase
	createGroup CreateGroupUseCase
	updateGroup UpdateGroupUseCase
}

func NewHandler(services Services) *Handler {
	return &Handler{
		createChat:  services.CreateChat,
		getChat:     services.GetChat,
		listChats:   services.ListChats,
		deleteChat:  services.DeleteChat,
		createGroup: services.CreateGroup,
		updateGroup: services.UpdateGroup,
	}
}

type createChatRequest struct {
	ContactID string `json:"contact_id"`
}

type chatResponse struct {
	ID           chat.ID       `json:"id"`
	Type         chat.Type     `json:"type"`
	Title        string        `json:"title,omitempty"`
	OwnerID      chat.UserID   `json:"owner_id,omitempty"`
	Participants []chat.UserID `json:"participants"`
	CreatedAt    string        `json:"created_at"`
}

func toChatResponse(c chat.Chat) chatResponse {
	return chatResponse{
		ID:           c.ID(),
		Type:         c.Type(),
		Title:        c.Title(),
		OwnerID:      c.Owner(),
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

	var req createChatRequest
	if err := decodeJSON(w, r, &req); err != nil {
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

func (h *Handler) DeleteChat(w http.ResponseWriter, r *http.Request) {
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

	if err := h.deleteChat.Execute(r.Context(), chatID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type participantResponse struct {
	UserID chat.UserID `json:"user_id"`
	Role   string      `json:"role"`
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
		role := "member"
		if c.Type() == chat.TypeGroup && p == c.Owner() {
			role = "owner"
		}
		result = append(result, participantResponse{UserID: p, Role: role})
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
	case errors.Is(err, usecase.ErrBlocked):
		writeError(w, http.StatusForbidden, "user is blocked")
	case errors.Is(err, chat.ErrInvalidParticipants),
		errors.Is(err, chat.ErrInvalidUserID),
		errors.Is(err, chat.ErrInvalidChatID):
		writeError(w, http.StatusBadRequest, "invalid chat request")
	case errors.Is(err, chat.ErrInvalidTitle),
		errors.Is(err, chat.ErrTooManyParticipants),
		errors.Is(err, chat.ErrCannotRemoveOwner):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, chat.ErrNotOwner):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, chat.ErrNotParticipant):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, chat.ErrAlreadyParticipant):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, chat.ErrUnsupportedChatType), errors.Is(err, chat.ErrNotGroup):
		writeError(w, http.StatusUnprocessableEntity, "unsupported chat type")
	default:
		logger.FromContext(r.Context()).Error("chat request failed", logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing data")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}
