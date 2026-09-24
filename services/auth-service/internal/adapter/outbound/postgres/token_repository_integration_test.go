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

const migrationsDir = "../../../../migrations"

func TestTokenRepository_SaveAndFindRefresh(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)

	queries := db.New(pg.Pool)
	tokenRepo := NewTokenRepository(queries)
	userRepo := NewUserRepository(queries)

	user := testUser()
	require.NoError(t, userRepo.Save(ctx, user))

	entry := testRefreshEntry(user.ID)
	require.NoError(t, tokenRepo.SaveRefresh(ctx, entry))

	got, err := tokenRepo.FindRefresh(ctx, entry.Token)
	require.NoError(t, err)

	require.Equal(t, entry.ID, got.ID)
	require.Equal(t, entry.UserID, got.UserID)
	require.Equal(t, entry.Token, got.Token)
	require.True(t, entry.CreatedAt.Equal(got.CreatedAt))
	require.True(t, entry.ExpiresAt.Equal(got.ExpiresAt))
}

func TestTokenRepository_FindRefresh_NotFound(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)

	repo := NewTokenRepository(db.New(pg.Pool))

	got, err := repo.FindRefresh(ctx, auth.RefreshToken("missing-token"))

	require.ErrorIs(t, err, auth.ErrRefreshInvalid)
	require.Empty(t, got)
}

func TestTokenRepository_DeleteRefresh(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)

	queries := db.New(pg.Pool)
	tokenRepo := NewTokenRepository(queries)
	userRepo := NewUserRepository(queries)

	user := testUser()
	require.NoError(t, userRepo.Save(ctx, user))

	entry := testRefreshEntry(user.ID)
	require.NoError(t, tokenRepo.SaveRefresh(ctx, entry))

	require.NoError(t, tokenRepo.DeleteRefresh(ctx, entry.Token))

	got, err := tokenRepo.FindRefresh(ctx, entry.Token)
	require.ErrorIs(t, err, auth.ErrRefreshInvalid)
	require.Empty(t, got)
}

func TestTokenRepository_DeleteRefresh_NotFound(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)

	repo := NewTokenRepository(db.New(pg.Pool))

	err := repo.DeleteRefresh(ctx, auth.RefreshToken("missing-token"))

	require.NoError(t, err)
}

func TestTokenRepository_SaveRefresh_MissingUser(t *testing.T) {
	ctx := context.Background()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)

	repo := NewTokenRepository(db.New(pg.Pool))

	entry := testRefreshEntry(auth.UserID("missing-user"))

	err := repo.SaveRefresh(ctx, entry)

	require.ErrorIs(t, err, outbound.ErrPersistence)
}

func testUser() auth.User {
	return auth.User{
		ID:           auth.UserID("user-1"),
		Email:        "user@example.com",
		PasswordHash: auth.PasswordHash("password-hash"),
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func testRefreshEntry(userID auth.UserID) auth.RefreshEntry {
	return auth.RefreshEntry{
		ID:        "entry-1",
		UserID:    userID,
		Token:     auth.RefreshToken("token.token.token"),
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}
