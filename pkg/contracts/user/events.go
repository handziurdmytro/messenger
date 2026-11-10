package user

import (
	"errors"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

const (
	SubjectCreated    = "user.created"
	SubjectCreatedDLQ = "user.created.dlq"
	SubjectDeleted    = "user.deleted"
	SubjectDeletedDLQ = "user.deleted.dlq"
)

var (
	ErrInvalidCreatedEvent = errors.New("invalid user created event")
	ErrInvalidDeletedEvent = errors.New("invalid user deleted event")
)

type CreatedEvent struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (e CreatedEvent) Validate() error {
	if _, err := uuid.Parse(e.UserID); err != nil {
		return ErrInvalidCreatedEvent
	}
	if _, err := mail.ParseAddress(e.Email); err != nil {
		return ErrInvalidCreatedEvent
	}
	if e.CreatedAt.IsZero() {
		return ErrInvalidCreatedEvent
	}
	return nil
}

type DeletedEvent struct {
	UserID    string    `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func (e DeletedEvent) Validate() error {
	if _, err := uuid.Parse(e.UserID); err != nil {
		return ErrInvalidDeletedEvent
	}
	if e.DeletedAt.IsZero() {
		return ErrInvalidDeletedEvent
	}
	return nil
}
