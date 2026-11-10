package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type BlockRepository struct {
	q *db.Queries
}

func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{q: db.New(pool)}
}

func (r *BlockRepository) Apply(ctx context.Context, change usecase.BlockChange) error {
	userID, err := uuid.Parse(string(change.UserID))
	if err != nil {
		return chat.ErrInvalidUserID
	}
	blockedID, err := uuid.Parse(string(change.BlockedID))
	if err != nil {
		return chat.ErrInvalidUserID
	}

	if err := r.q.ApplyUserBlock(ctx, db.ApplyUserBlockParams{
		UserID:    userID,
		BlockedID: blockedID,
		Blocked:   change.Blocked,
		UpdatedAt: change.At,
	}); err != nil {
		return errkit.Wrap(usecase.ErrPersistence, "postgres: apply user block", err)
	}

	return nil
}

func (r *BlockRepository) ExistsBetween(ctx context.Context, first, second chat.UserID) (bool, error) {
	firstID, err := uuid.Parse(string(first))
	if err != nil {
		return false, chat.ErrInvalidUserID
	}
	secondID, err := uuid.Parse(string(second))
	if err != nil {
		return false, chat.ErrInvalidUserID
	}

	exists, err := r.q.BlockExistsBetween(ctx, db.BlockExistsBetweenParams{FirstID: firstID, SecondID: secondID})
	if err != nil {
		return false, errkit.Wrap(usecase.ErrPersistence, "postgres: check user block", err)
	}

	return exists, nil
}
