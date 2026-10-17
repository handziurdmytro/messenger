package message

import (
	"encoding/json"
	"strings"
	"time"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
)

type Message struct {
	ID        ID
	ChatID    ChatID
	FromUser  string
	Type      Type
	Payload   json.RawMessage
	CreatedAt time.Time
	EditedAt  *time.Time
	DeletedAt *time.Time
}

func New(id ID, chatID ChatID, fromUser string, payload json.RawMessage, msgType Type) (Message, error) {
	if strings.TrimSpace(string(id)) == "" {
		return Message{}, ErrIDEmpty
	}
	if err := validateMessage(chatID, fromUser, msgType, payload); err != nil {
		return Message{}, err
	}

	return Message{
		ID:        id,
		ChatID:    chatID,
		FromUser:  fromUser,
		Type:      msgType,
		Payload:   clonePayload(payload),
		CreatedAt: time.Now().UTC(),
	}, nil
}

func Restore(id ID, chatID ChatID, fromUser string, msgType Type, payload json.RawMessage, createdAt time.Time, editedAt, deletedAt *time.Time) (Message, error) {
	if strings.TrimSpace(string(id)) == "" {
		return Message{}, ErrIDEmpty
	}
	if createdAt.IsZero() {
		return Message{}, ErrCreatedAtZero
	}
	if err := validateIdentity(chatID, fromUser, msgType); err != nil {
		return Message{}, err
	}

	if editedAt != nil && (!msgType.CanEdit() || editedAt.IsZero() || editedAt.Before(createdAt)) {
		return Message{}, ErrInvalidTimestamp
	}

	if deletedAt != nil {
		if !msgType.CanDelete() {
			return Message{}, ErrMessageNotDeletable
		}
		if deletedAt.IsZero() || deletedAt.Before(createdAt) || (editedAt != nil && deletedAt.Before(*editedAt)) {
			return Message{}, ErrInvalidTimestamp
		}
		if len(payload) != 0 {
			return Message{}, ErrPayloadTypeMismatch
		}
	} else if err := validatePayload(msgType, payload); err != nil {
		return Message{}, err
	}

	return Message{
		ID:        id,
		ChatID:    chatID,
		FromUser:  fromUser,
		Type:      msgType,
		Payload:   clonePayload(payload),
		CreatedAt: createdAt.UTC(),
		EditedAt:  cloneTime(editedAt),
		DeletedAt: cloneTime(deletedAt),
	}, nil
}

func (m *Message) Edit(newPayload json.RawMessage, now time.Time) error {
	if m.DeletedAt != nil {
		return ErrMessageDeleted
	}
	if !m.Type.CanEdit() {
		return ErrMessageNotEditable
	}
	if err := validatePayload(m.Type, newPayload); err != nil {
		return err
	}
	if err := m.validateChangeTime(now); err != nil {
		return err
	}

	m.Payload = clonePayload(newPayload)
	t := now.UTC()
	m.EditedAt = &t
	return nil
}

func (m *Message) Delete(now time.Time) error {
	if m.DeletedAt != nil {
		return ErrMessageDeleted
	}
	if !m.Type.CanDelete() {
		return ErrMessageNotDeletable
	}
	if err := m.validateChangeTime(now); err != nil {
		return err
	}

	m.Payload = nil
	t := now.UTC()
	m.DeletedAt = &t
	return nil
}

func (m *Message) validateChangeTime(now time.Time) error {
	if now.IsZero() || now.Before(m.CreatedAt) || (m.EditedAt != nil && now.Before(*m.EditedAt)) {
		return ErrInvalidTimestamp
	}
	return nil
}

func validateMessage(chatID ChatID, fromUser string, msgType Type, payload json.RawMessage) error {
	if err := validateIdentity(chatID, fromUser, msgType); err != nil {
		return err
	}
	return validatePayload(msgType, payload)
}

func validateIdentity(chatID ChatID, fromUser string, msgType Type) error {
	if strings.TrimSpace(string(chatID)) == "" {
		return ErrChatIDEmpty
	}
	if strings.TrimSpace(fromUser) == "" {
		return ErrFromUserEmpty
	}
	if !msgType.Valid() {
		return ErrTypeInvalid
	}
	return nil
}

func validatePayload(msgType Type, payload json.RawMessage) error {
	return contract.ValidatePayload(contract.Type(msgType), payload)
}

func clonePayload(payload json.RawMessage) json.RawMessage {
	if payload == nil {
		return nil
	}
	return append(json.RawMessage(nil), payload...)
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	tCopy := *t
	return &tCopy
}
