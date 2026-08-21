package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type UserCreatedConsumer struct {
	reader *kafka.Reader
	users  outbound.UserRepository
	log    *slog.Logger
}

func NewUserCreatedConsumer(
	brokers []string,
	topic, groupID string,
	users outbound.UserRepository,
	baseLogger *slog.Logger,
) *UserCreatedConsumer {
	return &UserCreatedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		users: users,
		log:   logger.WithComponent(baseLogger, "user-consumer"),
	}
}

type userCreatedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (c *UserCreatedConsumer) Start(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err = c.handle(ctx, msg); err != nil {
			c.log.Error("handle failed", logger.KeyError, err)
			continue
		}

		if err = c.reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("commit failed", logger.KeyError, err)
		}
	}
}

func (c *UserCreatedConsumer) handle(ctx context.Context, msg kafka.Message) error {
	var event userCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}

	u := user.NewFromEvent(event.UserID, event.Email, createdAt)
	return c.users.Save(ctx, u)
}

func (c *UserCreatedConsumer) Close() error {
	return c.reader.Close()
}
