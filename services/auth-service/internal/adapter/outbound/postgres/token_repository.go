package postgres

import (
	"context"
	"errors"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/jackc/pgx/v5"
)

type TokenRepository struct {
	q *db.Queries
}

func NewTokenRepository(q *db.Queries) *TokenRepository {
	return &TokenRepository{q: q}
}

func (r *TokenRepository) SaveRefresh(
	ctx context.Context,
	entry auth.RefreshEntry,
) error {
	err := r.q.CreateRefreshToken(
		ctx,
		db.CreateRefreshTokenParams{
			ID:        entry.ID,
			UserID:    string(entry.UserID),
			Token:     string(entry.Token),
			ExpiresAt: entry.ExpiresAt,
			CreatedAt: entry.CreatedAt,
		},
	)
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "create refresh token", err)
	}

	return nil
}

func (r *TokenRepository) FindRefresh(
	ctx context.Context,
	token auth.RefreshToken,
) (auth.RefreshEntry, error) {
	row, err := r.q.GetRefreshToken(
		ctx,
		string(token),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.RefreshEntry{},
			auth.ErrRefreshInvalid
	}
	if err != nil {
		return auth.RefreshEntry{},
			errkit.Wrap(outbound.ErrPersistence, "find refresh token", err)
	}

	return auth.RefreshEntry{
		ID:        row.ID,
		UserID:    auth.UserID(row.UserID),
		Token:     auth.RefreshToken(row.Token),
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *TokenRepository) DeleteRefresh(
	ctx context.Context,
	token auth.RefreshToken,
) error {
	err := r.q.DeleteRefreshToken(
		ctx,
		string(token),
	)
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "delete refresh token", err)
	}

	return nil
}
