package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

const maxUploadBody = usecase.MaxFileSize + (1 << 20)

type GetHistoryUseCase interface {
	Execute(ctx context.Context, q usecase.HistoryQuery) ([]message.Message, error)
}

type GetMessageUseCase interface {
	Execute(ctx context.Context, chatID message.ChatID, messageID message.ID) (message.Message, error)
}

type UploadFileUseCase interface {
	Execute(ctx context.Context, in usecase.UploadFileInput) error
}

type DownloadFileUseCase interface {
	Execute(ctx context.Context, key string) (io.Reader, error)
}

type EditMessageUseCase interface {
	Execute(ctx context.Context, in usecase.EditMessageInput) (message.Message, error)
}

type DeleteMessageUseCase interface {
	Execute(ctx context.Context, chatID message.ChatID, messageID message.ID, userID string) error
}

type Services struct {
	GetHistory    GetHistoryUseCase
	GetMessage    GetMessageUseCase
	UploadFile    UploadFileUseCase
	DownloadFile  DownloadFileUseCase
	EditMessage   EditMessageUseCase
	DeleteMessage DeleteMessageUseCase
	MarkRead      MarkReadUseCase
	ReadMarkers   ReadMarkersUseCase
	UnreadCount   UnreadCountUseCase
}

type Handler struct {
	history  GetHistoryUseCase
	get      GetMessageUseCase
	upload   UploadFileUseCase
	download DownloadFileUseCase
	edit     EditMessageUseCase
	remove   DeleteMessageUseCase
	markRead MarkReadUseCase
	markers  ReadMarkersUseCase
	unread   UnreadCountUseCase
}

func NewHandler(services Services) *Handler {
	return &Handler{
		history:  services.GetHistory,
		get:      services.GetMessage,
		upload:   services.UploadFile,
		download: services.DownloadFile,
		edit:     services.EditMessage,
		remove:   services.DeleteMessage,
		markRead: services.MarkRead,
		markers:  services.ReadMarkers,
		unread:   services.UnreadCount,
	}
}

type messageResponse struct {
	ID        message.ID      `json:"message_id"`
	ChatID    message.ChatID  `json:"chat_id"`
	FromUser  string          `json:"from_user"`
	Type      message.Type    `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	EditedAt  *time.Time      `json:"edited_at,omitempty"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

func toMessageResponse(m message.Message) messageResponse {
	return messageResponse{
		ID: m.ID, ChatID: m.ChatID, FromUser: m.FromUser, Type: m.Type,
		Payload: m.Payload, CreatedAt: m.CreatedAt, EditedAt: m.EditedAt, DeletedAt: m.DeletedAt,
	}
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	q := usecase.HistoryQuery{ChatID: message.ChatID(r.PathValue("chat_id"))}

	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before")
			return
		}
		q.Before = t
	}

	q.BeforeID = message.ID(r.URL.Query().Get("before_id"))

	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		q.Limit = n
	}

	messages, err := h.history.Execute(r.Context(), q)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	result := make([]messageResponse, 0, len(messages))
	for _, m := range messages {
		result = append(result, toMessageResponse(m))
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	m, err := h.get.Execute(
		r.Context(),
		message.ChatID(r.PathValue("chat_id")),
		message.ID(r.PathValue("message_id")),
	)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toMessageResponse(m))
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart file")
		return
	}
	defer file.Close()
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}

	key := uuid.NewString()
	err = h.upload.Execute(r.Context(), usecase.UploadFileInput{
		Key:         key,
		File:        file,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, struct {
		FileID string `json:"file_id"`
	}{FileID: key})
}

func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid file ID")
		return
	}

	file, err := h.download.Execute(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	if closer, ok := file.(io.Closer); ok {
		defer closer.Close()
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", id))

	if _, err := io.Copy(w, file); err != nil {
		logger.FromContext(r.Context()).Error("download file failed", "error", err)
	}
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidChatID),
		errors.Is(err, usecase.ErrInvalidMessageID),
		errors.Is(err, usecase.ErrInvalidHistoryCursor),
		errors.Is(err, usecase.ErrInvalidHistoryLimit),
		errors.Is(err, usecase.ErrEmptyKey),
		errors.Is(err, usecase.ErrEmptyFile),
		errors.Is(err, usecase.ErrInvalidFileSize),
		errors.Is(err, usecase.ErrUnsupportedMIME):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, usecase.ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())

	case errors.Is(err, usecase.ErrMessageNotFound):
		writeError(w, http.StatusNotFound, "message not found")

	case errors.Is(err, usecase.ErrForbidden):
		writeError(w, http.StatusForbidden, "only the author can change the message")

	case errors.Is(err, contract.ErrInvalidPayload):
		writeError(w, http.StatusBadRequest, "invalid payload")

	case errors.Is(err, message.ErrMessageDeleted):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, message.ErrMessageNotEditable),
		errors.Is(err, message.ErrMessageNotDeletable):
		writeError(w, http.StatusUnprocessableEntity, err.Error())

	default:
		logger.FromContext(r.Context()).Error("message REST request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

type editMessageRequest struct {
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req editMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.edit.Execute(r.Context(), usecase.EditMessageInput{
		ChatID:    message.ChatID(r.PathValue("chat_id")),
		MessageID: message.ID(r.PathValue("message_id")),
		UserID:    r.Header.Get("X-User-ID"),
		Payload:   req.Payload,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toMessageResponse(msg))
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	err := h.remove.Execute(
		r.Context(),
		message.ChatID(r.PathValue("chat_id")),
		message.ID(r.PathValue("message_id")),
		r.Header.Get("X-User-ID"),
	)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
