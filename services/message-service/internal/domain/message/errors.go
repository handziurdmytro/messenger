package message

import "errors"

var (
	ErrMessageEmpty        = errors.New("message body is empty")
	ErrChatIDEmpty         = errors.New("chat id is empty")
	ErrFromUserEmpty       = errors.New("from user is empty")
	ErrIDEmpty             = errors.New("message id is empty")
	ErrTypeInvalid         = errors.New("message type is invalid")
	ErrPayloadTypeMismatch = errors.New("payload does not match message type")
	ErrFileIDEmpty         = errors.New("file id is empty")
	ErrCreatedAtZero       = errors.New("created at is zero")
	ErrInvalidTimestamp    = errors.New("invalid timestamp")
	ErrMessageNotEditable  = errors.New("message is not editable")
	ErrMessageNotDeletable = errors.New("message is not deletable")
	ErrMessageDeleted      = errors.New("message is deleted")
)
