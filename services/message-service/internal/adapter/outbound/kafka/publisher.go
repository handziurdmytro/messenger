package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/segmentio/kafka-go"
)

type MessagePublisher struct {
	writer *kafka.Writer
}

func NewMessagePublisher(brokers []string, topic string) *MessagePublisher {
	return &MessagePublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			WriteTimeout: 5 * time.Second,
			RequiredAcks: kafka.RequireOne,
		},
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

func (p *MessagePublisher) Publish(ctx context.Context, msg message.Message) error {
	event := persistedEvent{
		MessageID: string(msg.ID),
		RoomID:    string(msg.RoomID),
		FromUser:  msg.FromUser,
		Type:      msg.Type,
		Body:      msg.Body,
		CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.RoomID),
		Value: payload,
	})
}

func (p *MessagePublisher) Close() error {
	return p.writer.Close()
}
