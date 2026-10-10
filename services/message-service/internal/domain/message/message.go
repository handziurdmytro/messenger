package message

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        ID
	RoomID    RoomID
	FromUser  string
	Type      Type
	Payload   Payload
	CreatedAt time.Time
	EditedAt  *time.Time
	DeletedAt *time.Time
}

func New(roomID RoomID, fromUser string, payload Payload, msgType Type) (Message, error) {
	if err := validateMessage(roomID, fromUser, msgType, payload); err != nil {
		return Message{}, err
	}

	return Message{
		ID:        ID(uuid.NewString()),
		RoomID:    roomID,
		FromUser:  fromUser,
		Type:      msgType,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func Restore(id ID, roomID RoomID, fromUser string, msgType Type, payload Payload, createdAt time.Time, editedAt, deletedAt *time.Time) (Message, error) {
	if strings.TrimSpace(string(id)) == "" {
		return Message{}, ErrIDEmpty
	}
	if createdAt.IsZero() {
		return Message{}, ErrCreatedAtZero
	}
	if err := validateIdentity(roomID, fromUser, msgType); err != nil {
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
		if payload != nil {
			return Message{}, ErrPayloadTypeMismatch
		}
	} else if err := validatePayload(msgType, payload); err != nil {
		return Message{}, err
	}

	return Message{
		ID:        id,
		RoomID:    roomID,
		FromUser:  fromUser,
		Type:      msgType,
		Payload:   payload,
		CreatedAt: createdAt,
		EditedAt:  cloneTime(editedAt),
		DeletedAt: cloneTime(deletedAt),
	}, nil
}

func (m *Message) Edit(newPayload Payload, now time.Time) error {
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

	m.Payload = newPayload
	t := now
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
	t := now
	m.DeletedAt = &t
	return nil
}

func (m *Message) validateChangeTime(now time.Time) error {
	if now.IsZero() || now.Before(m.CreatedAt) || (m.EditedAt != nil && now.Before(*m.EditedAt)) {
		return ErrInvalidTimestamp
	}
	return nil
}

func validateMessage(roomID RoomID, fromUser string, msgType Type, payload Payload) error {
	if err := validateIdentity(roomID, fromUser, msgType); err != nil {
		return err
	}
	return validatePayload(msgType, payload)
}

func validateIdentity(roomID RoomID, fromUser string, msgType Type) error {
	if strings.TrimSpace(string(roomID)) == "" {
		return ErrRoomIDEmpty
	}
	if strings.TrimSpace(fromUser) == "" {
		return ErrFromUserEmpty
	}
	if !msgType.Valid() {
		return ErrTypeInvalid
	}
	return nil
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	tCopy := *t
	return &tCopy
}
