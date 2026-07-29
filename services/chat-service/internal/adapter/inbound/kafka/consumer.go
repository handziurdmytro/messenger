package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/segmentio/kafka-go"
)

type PersistedConsumer struct {
	reader *kafka.Reader
	hub    *chat.Hub
	log    *slog.Logger
}

func NewPersistedConsumer(brokers []string, topic, groupID string, hub *chat.Hub, baseLogger *slog.Logger) *PersistedConsumer {
	return &PersistedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		hub: hub,
		log: logger.WithComponent(baseLogger, "persisted-consumer"),
	}
}

type persistedEvent struct {
	MessageID string `json:"message_id"`
	RoomID    string `json:"room_id"`
	FromUser  string `json:"from_user"`
	Type      string `json:"type"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (c *PersistedConsumer) Start(ctx context.Context) error {
	for {
		kafkaMsg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err = c.handle(ctx, kafkaMsg); err != nil {
			c.log.Error("handle failed", logger.KeyError, err)
		}

		if err = c.reader.CommitMessages(ctx, kafkaMsg); err != nil {
			c.log.Error("commit failed", logger.KeyError, err)
		}
	}
}

func (c *PersistedConsumer) handle(ctx context.Context, kafkaMsg kafka.Message) error {
	var event persistedEvent
	if err := json.Unmarshal(kafkaMsg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}

	confirm := chat.Message{
		ID:        event.MessageID,
		RoomID:    chat.RoomID(event.RoomID),
		FromUser:  event.FromUser,
		Type:      chat.MessageTypeConfirm,
		Body:      event.Body,
		CreatedAt: createdAt,
	}

	if err = c.hub.Confirm(event.FromUser, confirm); err != nil {
		c.log.Error("confirm failed", "message_id", event.MessageID, logger.KeyError, err)
	}

	return nil
}

func (c *PersistedConsumer) Close() error {
	return c.reader.Close()
}
