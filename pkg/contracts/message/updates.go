package message

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var (
	ErrInvalidEditedEvent  = errors.New("invalid message edited event")
	ErrInvalidDeletedEvent = errors.New("invalid message deleted event")
)

type EditedEvent struct {
	MessageID string          `json:"message_id"`
	ChatID    string          `json:"chat_id"`
	Type      Type            `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	EditedAt  time.Time       `json:"edited_at"`
}

func (e EditedEvent) Validate() error {
	if _, err := uuid.Parse(e.MessageID); err != nil {
		return errkit.Wrap(ErrInvalidEditedEvent, "parse message ID", err)
	}
	if _, err := uuid.Parse(e.ChatID); err != nil {
		return errkit.Wrap(ErrInvalidEditedEvent, "parse chat ID", err)
	}
	if e.EditedAt.IsZero() {
		return ErrInvalidEditedEvent
	}
	if err := ValidatePayload(e.Type, e.Payload); err != nil {
		return errkit.Wrap(ErrInvalidEditedEvent, "validate payload", err)
	}
	return nil
}

type DeletedEvent struct {
	MessageID string    `json:"message_id"`
	ChatID    string    `json:"chat_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func (e DeletedEvent) Validate() error {
	if _, err := uuid.Parse(e.MessageID); err != nil {
		return errkit.Wrap(ErrInvalidDeletedEvent, "parse message ID", err)
	}
	if _, err := uuid.Parse(e.ChatID); err != nil {
		return errkit.Wrap(ErrInvalidDeletedEvent, "parse chat ID", err)
	}
	if e.DeletedAt.IsZero() {
		return ErrInvalidDeletedEvent
	}
	return nil
}
