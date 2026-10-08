package contact

import "errors"

var (
	ErrAlreadyExists  = errors.New("contact already exists")
	ErrCannotAddSelf  = errors.New("cannot add yourself to contacts")
	ErrUserIDEmpty    = errors.New("user id is empty")
	ErrContactIDEmpty = errors.New("contact id is empty")
	ErrCreatedAtZero  = errors.New("created at is zero")
)
