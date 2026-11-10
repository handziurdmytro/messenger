package chat

import "errors"

var (
	ErrInvalidChatID       = errors.New("invalid chat id")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidParticipants = errors.New("invalid chat participants")
	ErrUnsupportedChatType = errors.New("unsupported chat type")
	ErrInvalidCreatedAt    = errors.New("invalid chat creation time")
	ErrInvalidTitle        = errors.New("invalid chat title")
	ErrTooManyParticipants = errors.New("too many chat participants")
	ErrNotGroup            = errors.New("chat is not a group")
	ErrNotOwner            = errors.New("only the owner can manage the group")
	ErrNotParticipant      = errors.New("user is not a participant")
	ErrAlreadyParticipant  = errors.New("user is already a participant")
	ErrCannotRemoveOwner   = errors.New("owner cannot be removed")
)
