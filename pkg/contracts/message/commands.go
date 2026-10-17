package message

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var (
	ErrInvalidMessageID   = errors.New("invalid message ID")
	ErrInvalidChatID      = errors.New("invalid chat ID")
	ErrInvalidSenderID    = errors.New("invalid sender ID")
	ErrInvalidMessageType = errors.New("invalid message type")
)

type SendMessageCommand struct {
	MessageID string          `json:"message_id"`
	ChatID    string          `json:"chat_id"`
	FromUser  string          `json:"from_user"`
	Type      Type            `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

func (c SendMessageCommand) Validate() error {
	if _, err := uuid.Parse(c.MessageID); err != nil {
		return errkit.Wrap(ErrInvalidMessageID, "parse message ID", err)
	}
	if _, err := uuid.Parse(c.ChatID); err != nil {
		return errkit.Wrap(ErrInvalidChatID, "parse chat ID", err)
	}
	if _, err := uuid.Parse(c.FromUser); err != nil {
		return errkit.Wrap(ErrInvalidSenderID, "parse sender ID", err)
	}
	if !c.Type.ClientSendable() {
		return ErrInvalidMessageType
	}

	return ValidatePayload(c.Type, c.Payload)
}
