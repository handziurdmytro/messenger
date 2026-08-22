package rest

import (
	"encoding/json"
	"net/http"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/pkg/logger"
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
	log := logger.WithComponent(logger.FromContext(r.Context()), "chat-rest")

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("create_room: invalid request body", logger.KeyError, err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoomID == "" {
		log.Warn("create_room: missing room_id")
		writeError(w, http.StatusBadRequest, "room_id is required")
		return
	}

	if err := h.hub.CreateRoom(chat.RoomID(req.RoomID)); err != nil {
		log.Error("create_room: failed", "room_id", req.RoomID, logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "failed to create room")
		return
	}

	log.Info("create_room: succeeded", "room_id", req.RoomID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createRoomResponse(req))
}
