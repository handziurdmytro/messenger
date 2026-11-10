package postgres

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const (
	blocksPrimaryKeyConstraint  = "blocks_pkey"
	blocksBlockedIDFKConstraint = "blocks_blocked_id_fk"
)

type BlockRepository struct {
	q *db.Queries
}

func NewBlockRepository(q *db.Queries) *BlockRepository {
	return &BlockRepository{q: q}
}

func (r *BlockRepository) Add(ctx context.Context, b block.Block) error {
	err := r.q.AddBlock(ctx, db.AddBlockParams{
		UserID:    b.UserID().String(),
		BlockedID: b.BlockedID().String(),
		CreatedAt: b.CreatedAt(),
	})
	switch {
	case err == nil:
		return nil
	case pgkit.IsConstraint(err, pgkit.CodeUniqueViolation, blocksPrimaryKeyConstraint):
		return block.ErrAlreadyBlocked
	case pgkit.IsConstraint(err, pgkit.CodeForeignKeyViolation, blocksBlockedIDFKConstraint):
		return user.ErrNotFound
	default:
		return errkit.Wrap(outbound.ErrPersistence, "add block", err)
	}
}

func (r *BlockRepository) Remove(ctx context.Context, userID, blockedID user.ID) error {
	rows, err := r.q.RemoveBlock(ctx, db.RemoveBlockParams{
		UserID:    userID.String(),
		BlockedID: blockedID.String(),
	})
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "remove block", err)
	}

	if rows == 0 {
		return block.ErrNotFound
	}

	return nil
}

func (r *BlockRepository) ListBlocked(ctx context.Context, userID user.ID, page outbound.Page) ([]user.User, error) {
	limit, offset, err := pageBounds(page)
	if err != nil {
		return nil, fmt.Errorf("list blocked users: %w", err)
	}

	rows, err := r.q.ListBlockedUsers(ctx, db.ListBlockedUsersParams{
		UserID:      userID.String(),
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "list blocked users", err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		u, err := toDomainUser(row)
		if err != nil {
			return nil, fmt.Errorf("restore blocked user %s: %w", row.ID, err)
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *BlockRepository) ExistsBetween(ctx context.Context, first, second user.ID) (bool, error) {
	exists, err := r.q.BlockExistsBetween(ctx, db.BlockExistsBetweenParams{
		FirstID:  first.String(),
		SecondID: second.String(),
	})
	if err != nil {
		return false, errkit.Wrap(outbound.ErrPersistence, "check block", err)
	}

	return exists, nil
}
