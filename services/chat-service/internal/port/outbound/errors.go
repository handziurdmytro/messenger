package outbound

import "errors"

var (
	ErrPersistence       = errors.New("persistence error")
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
)
