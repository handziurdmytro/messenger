package chat

import (
	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type ID string
type UserID string

func NewID() ID {
	return ID(uuid.NewString())
}

func ParseID(raw string) (ID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", errkit.Wrap(ErrInvalidChatID, "parse chat id", err)
	}
	return ID(id.String()), nil
}

func ParseUserID(raw string) (UserID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", errkit.Wrap(ErrInvalidUserID, "parse user id", err)
	}
	return UserID(id.String()), nil
}
