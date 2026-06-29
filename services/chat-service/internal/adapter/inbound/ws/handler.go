package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

const (
	writeTimeout   = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	/*todo: whitelist*/
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub *chat.Hub
}

func NewHandler(hub *chat.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "missing user id", http.StatusUnauthorized)
		return
	}

	roomID := chat.RoomID(r.URL.Query().Get("room_id"))
	if roomID == "" {
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws: upgrade failed", "error", err)
		return
	}

	client, err := h.hub.Join(userID, roomID)
	if err != nil {
		conn.Close()
		slog.Error("ws: join failed", "error", err)
		return
	}

	go h.writePump(conn, client)
	go h.readPump(conn, client)
}

func (h *Handler) readPump(conn *websocket.Conn, client *chat.Client) {
	defer func() {
		h.hub.Leave(client)
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Error("ws: unexpected close", "error", err)
			}
			return
		}

		msg, err := parseMessage(raw, client)
		if err != nil {
			slog.Error("ws: parse failed", "error", err)
			continue
		}

		if err = h.hub.Publish(msg); err != nil {
			slog.Error("ws: publish failed", "error", err)
		}
	}
}

func (h *Handler) writePump(conn *websocket.Conn, client *chat.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.Send():
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			raw, err := json.Marshal(msg)
			if err != nil {
				slog.Error("ws: marshal failed", "error", err)
				continue
			}

			if err = conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type incomingMessage struct {
	Body string `json:"body"`
	Type string `json:"type"`
}

func parseMessage(raw []byte, client *chat.Client) (chat.Message, error) {
	var incoming incomingMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return chat.Message{}, err
	}

	return chat.Message{
		RoomID:   client.RoomID(),
		FromUser: client.UserID(),
		Type:     chat.MessageType(incoming.Type),
		Body:     incoming.Body,
	}, nil
}
