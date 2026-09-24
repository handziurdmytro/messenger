//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/pgkit/pgtest"
	"github.com/stretchr/testify/require"
)

func TestTransactor_WithinTransaction_Commits(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	transactor := NewTransactor(pg.Pool)
	userRepo := NewUserRepository(db.New(pg.Pool))

	user := testTxUser()

	err := transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		return repos.Users.Save(ctx, user)
	})
	require.NoError(t, err)

	got, err := userRepo.FindByID(ctx, user.ID)
	require.NoError(t, err)

	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email, got.Email)
	require.Equal(t, user.PasswordHash, got.PasswordHash)
	require.True(t, user.CreatedAt.Equal(got.CreatedAt))
}

func TestTransactor_WithinTransaction_RollsBack(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	transactor := NewTransactor(pg.Pool)
	userRepo := NewUserRepository(db.New(pg.Pool))

	user := testTxUser()
	expectedErr := errors.New("rollback")

	err := transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		require.NoError(t, repos.Users.Save(ctx, user))
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)

	got, err := userRepo.FindByID(ctx, user.ID)
	require.ErrorIs(t, err, auth.ErrUserNotFound)
	require.Empty(t, got)
}

func TestTransactor_WithinTransaction_IsAtomic(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	transactor := NewTransactor(pg.Pool)

	queries := db.New(pg.Pool)
	userRepo := NewUserRepository(queries)
	tokenRepo := NewTokenRepository(queries)

	user := testTxUser()
	entry := auth.RefreshEntry{
		ID:        "entry-1",
		UserID:    user.ID,
		Token:     auth.RefreshToken("token.token.token"),
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	expectedErr := errors.New("rollback")

	err := transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		require.NoError(t, repos.Users.Save(ctx, user))
		require.NoError(t, repos.Tokens.SaveRefresh(ctx, entry))

		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)

	gotUser, err := userRepo.FindByID(ctx, user.ID)
	require.ErrorIs(t, err, auth.ErrUserNotFound)
	require.Empty(t, gotUser)

	gotEntry, err := tokenRepo.FindRefresh(ctx, entry.Token)
	require.ErrorIs(t, err, auth.ErrRefreshInvalid)
	require.Empty(t, gotEntry)
}

func testTxUser() auth.User {
	return auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("hash1234"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
