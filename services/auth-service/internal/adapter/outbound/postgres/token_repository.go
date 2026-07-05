package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

func (r *TokenRepository) SaveRefresh(ctx context.Context, entry auth.RefreshEntry) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at)
        VALUES ($1, $2, $3, $4, $5)`,
		entry.ID,
		string(entry.UserID),
		string(entry.Token),
		entry.ExpiresAt,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *TokenRepository) FindRefresh(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
	var entry auth.RefreshEntry
	var id, userID, tok string

	err := r.pool.QueryRow(ctx, `
        SELECT id, user_id, token, expires_at, created_at
        FROM refresh_tokens WHERE token = $1`,
		string(token),
	).Scan(&id, &userID, &tok, &entry.ExpiresAt, &entry.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return auth.RefreshEntry{}, auth.ErrRefreshInvalid
	}
	if err != nil {
		return auth.RefreshEntry{}, fmt.Errorf("find refresh token: %w", err)
	}

	entry.ID = id
	entry.UserID = auth.UserID(userID)
	entry.Token = auth.RefreshToken(tok)
	return entry, nil
}

func (r *TokenRepository) DeleteRefresh(ctx context.Context, token auth.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
        DELETE FROM refresh_tokens WHERE token = $1`,
		string(token),
	)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
