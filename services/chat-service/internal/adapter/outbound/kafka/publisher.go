package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
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

type messageEvent struct {
	RoomID    string `json:"room_id"`
	FromUser  string `json:"from_user"`
	Type      string `json:"type"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (p *MessagePublisher) Publish(ctx context.Context, msg chat.Message) error {
	event := messageEvent{
		RoomID:    string(msg.RoomID),
		FromUser:  msg.FromUser,
		Type:      string(msg.Type),
		Body:      msg.Body,
		CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal message event: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.RoomID),
		Value: payload,
	})
}

func (p *MessagePublisher) Close() error {
	return p.writer.Close()
}
