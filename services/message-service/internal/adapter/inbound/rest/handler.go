package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/port/outbound"
)

type Handler struct {
	repo outbound.MessageRepository
}

func NewHandler(repo outbound.MessageRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, "room_id is required")
		return
	}

	q := message.Query{RoomID: message.RoomID(roomID)}

	if before := r.URL.Query().Get("before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			q.Before = t
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			q.Limit = l
		}
	}

	messages, err := h.repo.GetHistory(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}
