package rest

import (
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
	"github.com/handziurdmytro/messenger/pkg/logger"
)

const maxUploadBody = usecase.MaxFileSize + (1 << 20)

type Handler struct {
	history  *usecase.GetHistory
	get      *usecase.GetMessage
	upload   *usecase.UploadFile
	download *usecase.DownloadFile
}

func NewHandler(
	history *usecase.GetHistory,
	get *usecase.GetMessage,
	upload *usecase.UploadFile,
	download *usecase.DownloadFile,
) *Handler {
	return &Handler{history: history, get: get, upload: upload, download: download}
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
		defer r.MultipartForm.RemoveAll()
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

	default:
		logger.FromContext(r.Context()).Error("message REST request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
