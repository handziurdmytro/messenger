package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type MessageConsumer struct {
	reader     *kafka.Reader
	repository outbound.MessageRepository
	publisher  outbound.MessagePublisher
	log        *slog.Logger
}

func NewMessageConsumer(
	brokers []string,
	topic string,
	groupID string,
	repository outbound.MessageRepository,
	publisher outbound.MessagePublisher,
	baseLogger *slog.Logger,
) *MessageConsumer {
	return &MessageConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		repository: repository,
		publisher:  publisher,
		log:        logger.WithComponent(baseLogger, "message-consumer"),
	}
}

type sentEvent struct {
	RoomID    string `json:"room_id"`
	FromUser  string `json:"from_user"`
	Type      string `json:"type"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (c *MessageConsumer) Start(ctx context.Context) error {
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
			continue
		}

		if err = c.reader.CommitMessages(ctx, kafkaMsg); err != nil {
			c.log.Error("commit failed", logger.KeyError, err)
		}
	}
}

func (c *MessageConsumer) handle(ctx context.Context, kafkaMsg kafka.Message) error {
	var event sentEvent
	if err := json.Unmarshal(kafkaMsg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	msg, err := message.New(
		message.RoomID(event.RoomID),
		event.FromUser,
		event.Body,
		event.Type,
	)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	if err = c.repository.Save(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	if err = c.publisher.Publish(ctx, msg); err != nil {
		c.log.Error("publish persisted event failed", "message_id", msg.ID, logger.KeyError, err)
	}

	return nil
}

func (c *MessageConsumer) Close() error {
	return c.reader.Close()
}
