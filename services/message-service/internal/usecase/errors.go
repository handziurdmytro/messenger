package usecase

import "errors"

var (
	ErrInternal        = errors.New("internal error")
	ErrEmptyKey        = errors.New("file key is empty")
	ErrEmptyFile       = errors.New("file is empty")
	ErrFileTooLarge    = errors.New("file is too large")
	ErrInvalidFileSize = errors.New("invalid file size")
	ErrUnsupportedMIME = errors.New("unsupported MIME type")
)
