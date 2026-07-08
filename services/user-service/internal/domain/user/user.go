package user

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrDisplayNameEmpty = errors.New("display name is empty")
)

type ID string

type User struct {
	ID          ID
	Email       string
	DisplayName string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFromEvent(id, email string, createdAt time.Time) User {
	return User{
		ID:          ID(id),
		Email:       email,
		DisplayName: defaultDisplayName(email),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func defaultDisplayName(email string) string {
	for i, ch := range email {
		if ch == '@' {
			return email[:i]
		}
	}
	return email
}
