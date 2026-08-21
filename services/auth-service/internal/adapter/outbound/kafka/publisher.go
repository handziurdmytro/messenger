package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type EventPublisher struct {
	writer *kafka.Writer
}

func NewEventPublisher(brokers []string, topic string) *EventPublisher {
	return &EventPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			WriteTimeout: 5 * time.Second,
			RequiredAcks: kafka.RequireOne,
		},
	}
}

type userCreatedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (p *EventPublisher) PublishUserCreated(ctx context.Context, user auth.User) error {
	event := userCreatedEvent{
		UserID:    string(user.ID),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(user.ID),
		Value: payload,
	})
}

func (p *EventPublisher) Close() error {
	return p.writer.Close()
}
