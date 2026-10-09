package user

import "errors"

var (
	ErrNotFound         = errors.New("user not found")
	ErrAlreadyExists    = errors.New("user already exists")
	ErrEmailTaken       = errors.New("email already taken")
	ErrEmailInvalid     = errors.New("email is invalid")
	ErrIDEmpty          = errors.New("user id is empty")
	ErrEmailEmpty       = errors.New("email is empty")
	ErrDisplayNameEmpty = errors.New("display name is empty")
	ErrCreatedAtZero    = errors.New("created at is zero")
	ErrUpdatedAtZero    = errors.New("updated at is zero")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)
