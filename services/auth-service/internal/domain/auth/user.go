package auth

import (
	"errors"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailEmpty   = errors.New("email is empty")
	ErrEmailInvalid = errors.New("email format is invalid")
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

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil {
		return User{}, ErrEmailInvalid
	}

	hash, err := HashPassword(plainPassword)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           UserID(uuid.New().String()),
		Email:        parsedEmail.Address,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
