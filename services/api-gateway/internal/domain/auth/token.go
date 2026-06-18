package auth

import (
	"errors"
	"time"
)

var (
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenInvalid   = errors.New("token invalid")
	ErrTokenMalformed = errors.New("token malformed")
)

type Token string

type Claims struct {
	UserID    string
	Email     string
	Roles     []Role
	ExpiresAt time.Time
	IssuedAt  time.Time
}

func (c Claims) ToIdentity() Identity {
	return Identity{
		UserID:    c.UserID,
		Email:     c.Email,
		Roles:     c.Roles,
		ExpiresAt: c.ExpiresAt,
	}
}
