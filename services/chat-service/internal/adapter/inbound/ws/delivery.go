package ws

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var ErrDeliveryFailed = errors.New("websocket delivery failed")

type MessageDelivery struct {
	chats    usecase.ChatRepository
	registry *Registry
}

func NewMessageDelivery(chats usecase.ChatRepository, registry *Registry) *MessageDelivery {
	return &MessageDelivery{
		chats:    chats,
		registry: registry,
	}
}

func (d *MessageDelivery) Deliver(ctx context.Context, event message.CreatedEvent) error {
	return d.send(ctx, event.ChatID, "message.created", event)
}

func (d *MessageDelivery) DeliverEdited(ctx context.Context, event message.EditedEvent) error {
	return d.send(ctx, event.ChatID, "message.edited", event)
}

func (d *MessageDelivery) DeliverDeleted(ctx context.Context, event message.DeletedEvent) error {
	return d.send(ctx, event.ChatID, "message.deleted", event)
}

func (d *MessageDelivery) send(ctx context.Context, rawChatID, eventType string, event any) error {
	chatID, err := chat.ParseID(rawChatID)
	if err != nil {
		return errkit.Wrap(ErrDeliveryFailed, "parse chat ID", err)
	}

	c, err := d.chats.FindByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, usecase.ErrChatNotFound) {
			return nil
		}
		return errkit.Wrap(ErrDeliveryFailed, "find message chat", err)
	}

	data, err := json.Marshal(struct {
		Type    string `json:"type"`
		Message any    `json:"message"`
	}{
		Type:    eventType,
		Message: event,
	})
	if err != nil {
		return errkit.Wrap(ErrDeliveryFailed, "marshal message event", err)
	}

	d.registry.SendTo(c.Participants(), data)
	return nil
}
