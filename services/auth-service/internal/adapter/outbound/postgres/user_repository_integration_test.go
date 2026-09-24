//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/pgkit/pgtest"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_SaveAndFindByEmail(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	user := auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("password-hash"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	require.NoError(t, repo.Save(ctx, user))

	got, err := repo.FindByEmail(ctx, user.Email)
	require.NoError(t, err)

	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email, got.Email)
	require.Equal(t, user.PasswordHash, got.PasswordHash)
	require.True(t, user.CreatedAt.Equal(got.CreatedAt))
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	got, err := repo.FindByEmail(ctx, "missing@example.com")

	require.ErrorIs(t, err, auth.ErrUserNotFound)
	require.Empty(t, got)
}

func TestUserRepository_SaveAndFindByID(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	user := auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("password-hash"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	require.NoError(t, repo.Save(ctx, user))

	got, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)

	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email, got.Email)
	require.Equal(t, user.PasswordHash, got.PasswordHash)
	require.True(t, user.CreatedAt.Equal(got.CreatedAt))
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	got, err := repo.FindByID(ctx, auth.UserID("missing"))

	require.ErrorIs(t, err, auth.ErrUserNotFound)
	require.Empty(t, got)
}

func TestUserRepository_Save_DuplicateID(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	user := auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("password-hash"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	require.NoError(t, repo.Save(ctx, user))

	err := repo.Save(ctx, user)

	require.ErrorIs(t, err, outbound.ErrPersistence)
}

func TestUserRepository_Save_DuplicateEmail(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	repo := NewUserRepository(db.New(pg.Pool))

	first := auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("password-hash"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	second := first
	second.ID = auth.UserID("user-2")

	require.NoError(t, repo.Save(ctx, first))

	err := repo.Save(ctx, second)

	require.ErrorIs(t, err, auth.ErrEmailTaken)
}
