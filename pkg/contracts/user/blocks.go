package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	SubjectBlocked   = "user.blocked"
	SubjectUnblocked = "user.unblocked"
)

var ErrInvalidBlockEvent = errors.New("invalid user block event")

type BlockEvent struct {
	UserID     string    `json:"user_id"`
	BlockedID  string    `json:"blocked_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e BlockEvent) Validate() error {
	if _, err := uuid.Parse(e.UserID); err != nil {
		return ErrInvalidBlockEvent
	}
	if _, err := uuid.Parse(e.BlockedID); err != nil {
		return ErrInvalidBlockEvent
	}
	if e.UserID == e.BlockedID || e.OccurredAt.IsZero() {
		return ErrInvalidBlockEvent
	}
	return nil
}
