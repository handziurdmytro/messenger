package rest

import (
	"encoding/json"
	"net/http"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type Handler struct {
	hub *chat.Hub
}

func NewHandler(hub *chat.Hub) *Handler {
	return &Handler{hub: hub}
}

type createRoomRequest struct {
	RoomID string `json:"room_id"`
}

type createRoomResponse struct {
	RoomID string `json:"room_id"`
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoomID == "" {
		writeError(w, http.StatusBadRequest, "room_id is required")
		return
	}

	if err := h.hub.CreateRoom(chat.RoomID(req.RoomID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createRoomResponse{RoomID: req.RoomID})
}
