package usecase

import "errors"

var (
	ErrPersistence       = errors.New("persistence error")
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
	ErrForbidden         = errors.New("access denied")
	ErrPublishFailed     = errors.New("failed to publish a message")
	ErrConsumeFailed     = errors.New("message consumption failed")
)
