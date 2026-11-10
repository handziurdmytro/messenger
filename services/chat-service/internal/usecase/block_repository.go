package usecase

import (
	"context"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type BlockChange struct {
	UserID    chat.UserID
	BlockedID chat.UserID
	Blocked   bool
	At        time.Time
}

type BlockRepository interface {
	Apply(ctx context.Context, change BlockChange) error
	ExistsBetween(ctx context.Context, first, second chat.UserID) (bool, error)
}
