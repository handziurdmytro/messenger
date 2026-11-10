package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
)

type MarkReadUseCase interface {
	Execute(ctx context.Context, chatID message.ChatID, messageID message.ID, userID string) (usecase.ReadMarker, error)
}

type ReadMarkersUseCase interface {
	Execute(ctx context.Context, chatID message.ChatID) ([]usecase.ReadMarker, error)
}

type UnreadCountUseCase interface {
	Execute(ctx context.Context, chatID message.ChatID, userID string) (usecase.UnreadCount, error)
}

type markReadRequest struct {
	MessageID message.ID `json:"message_id"`
}

type readMarkerResponse struct {
	UserID    string     `json:"user_id"`
	MessageID message.ID `json:"message_id"`
	ReadAt    time.Time  `json:"read_at"`
}

type unreadCountResponse struct {
	Count  int  `json:"count"`
	Capped bool `json:"capped"`
}

func toReadMarkerResponse(m usecase.ReadMarker) readMarkerResponse {
	return readMarkerResponse{UserID: m.UserID, MessageID: m.MessageID, ReadAt: m.ReadAt}
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)

	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	marker, err := h.markRead.Execute(
		r.Context(),
		message.ChatID(r.PathValue("chat_id")),
		req.MessageID,
		r.Header.Get("X-User-ID"),
	)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toReadMarkerResponse(marker))
}

func (h *Handler) ListReadMarkers(w http.ResponseWriter, r *http.Request) {
	markers, err := h.markers.Execute(r.Context(), message.ChatID(r.PathValue("chat_id")))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	result := make([]readMarkerResponse, 0, len(markers))
	for _, m := range markers {
		result = append(result, toReadMarkerResponse(m))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.unread.Execute(r.Context(), message.ChatID(r.PathValue("chat_id")), r.Header.Get("X-User-ID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, unreadCountResponse{Count: count.Count, Capped: count.Capped})
}
