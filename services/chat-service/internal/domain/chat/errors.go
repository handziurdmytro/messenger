package chat

import "errors"

var (
	ErrInvalidChatID       = errors.New("invalid chat id")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidParticipants = errors.New("invalid chat participants")
	ErrUnsupportedChatType = errors.New("unsupported chat type")
	ErrInvalidCreatedAt    = errors.New("invalid chat creation time")
)
