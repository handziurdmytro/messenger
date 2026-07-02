package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailEmpty   = errors.New("email is empty")
	ErrEmailTaken   = errors.New("email already taken")
	ErrUserNotFound = errors.New("user not found")
)

type UserID string

type User struct {
	ID           UserID
	Email        string
	PasswordHash PasswordHash
	CreatedAt    time.Time
}

func NewUser(email, plainPassword string) (User, error) {
	if email == "" {
		return User{}, ErrEmailEmpty
	}

	hash, err := HashPassword(plainPassword)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           UserID(uuid.New().String()),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
