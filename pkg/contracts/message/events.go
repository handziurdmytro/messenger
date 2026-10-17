package message

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var ErrInvalidCreatedEvent = errors.New("invalid message created event")

type CreatedEvent struct {
	MessageID string          `json:"message_id"`
	ChatID    string          `json:"chat_id"`
	FromUser  string          `json:"from_user"`
	Type      Type            `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func (e CreatedEvent) Validate() error {
	if _, err := uuid.Parse(e.MessageID); err != nil {
		return errkit.Wrap(ErrInvalidCreatedEvent, "parse message ID", err)
	}
	if _, err := uuid.Parse(e.ChatID); err != nil {
		return errkit.Wrap(ErrInvalidCreatedEvent, "parse chat ID", err)
	}
	if _, err := uuid.Parse(e.FromUser); err != nil {
		return errkit.Wrap(ErrInvalidCreatedEvent, "parse sender ID", err)
	}
	if e.CreatedAt.IsZero() {
		return ErrInvalidCreatedEvent
	}
	if !e.Type.ClientSendable() {
		return ErrInvalidCreatedEvent
	}

	if err := ValidatePayload(e.Type, e.Payload); err != nil {
		return errkit.Wrap(ErrInvalidCreatedEvent, "validate payload", err)
	}

	return nil
}
