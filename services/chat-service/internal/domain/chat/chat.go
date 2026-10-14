package chat

import (
	"slices"
	"time"
)

type Type string

const (
	TypePrivate Type = "private"
	TypeGroup   Type = "group"
)

type Chat struct {
	id           ID
	chatType     Type
	participants []UserID
	createdAt    time.Time
}

func NewPrivate(user1, user2 UserID) (Chat, error) {
	if err := validatePrivateParticipants(user1, user2); err != nil {
		return Chat{}, err
	}

	return Chat{
		id:           NewID(),
		chatType:     TypePrivate,
		participants: []UserID{user1, user2},
		createdAt:    time.Now().UTC(),
	}, nil
}

func Restore(id ID, chatType Type, participants []UserID, createdAt time.Time) (Chat, error) {
	if _, err := ParseID(string(id)); err != nil {
		return Chat{}, err
	}
	if createdAt.IsZero() {
		return Chat{}, ErrInvalidCreatedAt
	}

	switch chatType {
	case TypePrivate:
		if len(participants) != 2 {
			return Chat{}, ErrInvalidParticipants
		}
		if err := validatePrivateParticipants(participants[0], participants[1]); err != nil {
			return Chat{}, err
		}
	default:
		return Chat{}, ErrUnsupportedChatType
	}

	return Chat{
		id:           id,
		chatType:     chatType,
		participants: slices.Clone(participants),
		createdAt:    createdAt.UTC(),
	}, nil
}

func validatePrivateParticipants(user1, user2 UserID) error {
	if _, err := ParseUserID(string(user1)); err != nil {
		return err
	}
	if _, err := ParseUserID(string(user2)); err != nil {
		return err
	}
	if user1 == user2 {
		return ErrInvalidParticipants
	}
	return nil
}

func (c Chat) ID() ID {
	return c.id
}

func (c Chat) Type() Type {
	return c.chatType
}

func (c Chat) CreatedAt() time.Time {
	return c.createdAt
}

func (c Chat) HasParticipant(id UserID) bool {
	return slices.Contains(c.participants, id)
}

func (c Chat) Participants() []UserID {
	return slices.Clone(c.participants)
}
