package usecase

import "errors"

var (
	ErrInternal             = errors.New("internal error")
	ErrPersistence          = errors.New("persistence error")
	ErrPublishFailed        = errors.New("message event publish failed")
	ErrInvalidChatID        = errors.New("invalid chat ID")
	ErrInvalidHistoryCursor = errors.New("invalid history cursor")
	ErrInvalidHistoryLimit  = errors.New("invalid history limit")

	ErrEmptyKey         = errors.New("file key is empty")
	ErrEmptyFile        = errors.New("file is empty")
	ErrFileTooLarge     = errors.New("file is too large")
	ErrInvalidFileSize  = errors.New("invalid file size")
	ErrUnsupportedMIME  = errors.New("unsupported MIME type")
	ErrInvalidMessageID = errors.New("invalid message ID")
	ErrMessageNotFound  = errors.New("message not found")
)
