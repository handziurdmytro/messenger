package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
)

const (
	maxMessageSize     = 64 << 10
	sendMessageTimeout = 15 * time.Second
)

type Handler struct {
	registry    *Registry
	sendMessage *usecase.SendMessageUseCase
	logger      *slog.Logger
}

func NewHandler(registry *Registry, sendMessage *usecase.SendMessageUseCase, logger *slog.Logger) *Handler {
	return &Handler{
		registry:    registry,
		sendMessage: sendMessage,
		logger:      logger,
	}
}

type incomingMessage struct {
	MessageID string          `json:"message_id"`
	ChatID    string          `json:"chat_id"`
	Type      message.Type    `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type outgoingResponse struct {
	Type      string `json:"type"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, err := chat.ParseUserID(r.Header.Get("X-User-ID"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}

	conn.SetReadLimit(maxMessageSize)

	client := newConnection(userID, conn)
	if !h.registry.Add(client) {
		client.Close()
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	writeDone := make(chan struct{})

	go func() {
		defer close(writeDone)
		client.writeLoop(ctx)
	}()

	h.readLoop(ctx, client)

	cancel()
	h.registry.Remove(client)
	<-writeDone
}

func (h *Handler) readLoop(ctx context.Context, client *Connection) {
	for {
		typ, data, err := client.conn.Read(ctx)
		if err != nil {
			return
		}

		if typ != websocket.MessageText {
			h.respond(client, outgoingResponse{
				Type:  "error",
				Error: "expected text message",
			})
			continue
		}

		var incoming incomingMessage
		if err := json.Unmarshal(data, &incoming); err != nil {
			h.respond(client, outgoingResponse{
				Type:  "error",
				Error: "invalid message format",
			})
			continue
		}

		chatID, err := chat.ParseID(incoming.ChatID)
		if err != nil {
			h.respond(client, outgoingResponse{
				Type:      "error",
				MessageID: incoming.MessageID,
				Error:     "invalid chat ID",
			})
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, sendMessageTimeout)
		id, err := h.sendMessage.Execute(sendCtx, usecase.SendMessageInput{
			MessageID: incoming.MessageID,
			ChatID:    chatID,
			FromUser:  client.userID,
			Type:      incoming.Type,
			Payload:   incoming.Payload,
		})
		cancel()

		if err != nil {
			h.logger.Warn("send message failed",
				"message_id", incoming.MessageID,
				"error", err,
			)
			h.respond(client, outgoingResponse{
				Type:      "error",
				MessageID: incoming.MessageID,
				Error:     "failed to send message",
			})
			continue
		}

		h.respond(client, outgoingResponse{
			Type:      "message.accepted",
			MessageID: id,
		})
	}
}

func (h *Handler) respond(client *Connection, response outgoingResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("marshal websocket response failed", "error", err)
		return
	}

	if !client.trySend(data) {
		client.Close()
	}
}
