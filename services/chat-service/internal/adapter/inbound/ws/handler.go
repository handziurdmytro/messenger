package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/logger"
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
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

type Handler struct {
	hub       *chat.Hub
	publisher outbound.MessagePublisher
}

func NewHandler(hub *chat.Hub, publisher outbound.MessagePublisher) *Handler {
	return &Handler{
		hub:       hub,
		publisher: publisher,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "ws")

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		log.Warn("ws: missing user id")
		http.Error(w, "missing user id", http.StatusUnauthorized)
		return
	}

	roomID := chat.RoomID(r.URL.Query().Get("room_id"))
	if roomID == "" {
		log.Warn("ws: missing room_id", "user_id", userID)
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}

	connLog := log.With("user_id", userID, "room_id", string(roomID))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		connLog.Error("ws: upgrade failed", logger.KeyError, err)
		return
	}

	client, err := h.hub.Join(userID, roomID)
	if err != nil {
		conn.Close()
		connLog.Warn("ws: join failed", logger.KeyError, err)
		return
	}

	connLog.Info("ws: connected")

	go h.writePump(conn, client, connLog)
	go h.readPump(r.Context(), conn, client, connLog)
}

func (h *Handler) readPump(ctx context.Context, conn *websocket.Conn, client *chat.Client, log *slog.Logger) {
	defer func() {
		_ = h.hub.Leave(client)
		conn.Close()
		log.Info("ws: disconnected")
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
				log.Error("ws: unexpected close", logger.KeyError, err)
			}
			return
		}

		msg, err := parseMessage(raw, client)
		if err != nil {
			log.Warn("ws: invalid message payload", logger.KeyError, err)
			continue
		}

		if err := h.hub.Publish(msg); err != nil {
			log.Warn("ws: publish to room failed", logger.KeyError, err)
			continue
		}

		go func(m chat.Message) {
			publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := h.publisher.Publish(publishCtx, m); err != nil {
				log.Error("kafka: publish failed", logger.KeyError, err)
				// TODO: dead letter queue or retry
			}
		}(msg)
	}
}

func (h *Handler) writePump(conn *websocket.Conn, client *chat.Client, log *slog.Logger) {
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
				log.Error("ws: marshal failed", logger.KeyError, err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
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
