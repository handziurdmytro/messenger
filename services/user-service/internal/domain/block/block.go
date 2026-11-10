package block

import (
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type Block struct {
	userID    user.ID
	blockedID user.ID
	createdAt time.Time
}

func New(userID, blockedID user.ID, createdAt time.Time) (Block, error) {
	if userID.IsZero() {
		return Block{}, ErrUserIDEmpty
	}

	if blockedID.IsZero() {
		return Block{}, ErrBlockedIDEmpty
	}

	if userID == blockedID {
		return Block{}, ErrCannotBlockSelf
	}

	if createdAt.IsZero() {
		return Block{}, ErrCreatedAtZero
	}

	return Block{
		userID:    userID,
		blockedID: blockedID,
		createdAt: createdAt,
	}, nil
}

func (b Block) UserID() user.ID {
	return b.userID
}

func (b Block) BlockedID() user.ID {
	return b.blockedID
}

func (b Block) CreatedAt() time.Time {
	return b.createdAt
}
