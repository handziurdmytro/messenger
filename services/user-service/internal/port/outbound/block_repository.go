package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type BlockRepository interface {
	Add(ctx context.Context, b block.Block) error
	Remove(ctx context.Context, userID, blockedID user.ID) error
	ListBlocked(ctx context.Context, userID user.ID, page Page) ([]user.User, error)
	ExistsBetween(ctx context.Context, first, second user.ID) (bool, error)
}
