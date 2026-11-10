//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	"github.com/handziurdmytro/messenger/pkg/pgkit/pgtest"
)

func newAccountTestQueries(t *testing.T) *db.Queries {
	t.Helper()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, migrationsDir)
	return db.New(pg.Pool)
}

func TestTokenRepository_DeleteAllRefresh(t *testing.T) {
	ctx := context.Background()
	queries := newAccountTestQueries(t)
	tokens := NewTokenRepository(queries)
	users := NewUserRepository(queries)

	user := testUser()
	require.NoError(t, users.Save(ctx, user))

	first := testRefreshEntry(user.ID)
	second := testRefreshEntry(user.ID)
	second.ID = "entry-2"
	second.Token = "token.token.second"
	require.NoError(t, tokens.SaveRefresh(ctx, first))
	require.NoError(t, tokens.SaveRefresh(ctx, second))

	require.NoError(t, tokens.DeleteAllRefresh(ctx, user.ID))

	for _, entry := range []auth.RefreshEntry{first, second} {
		_, err := tokens.FindRefresh(ctx, entry.Token)
		require.ErrorIs(t, err, auth.ErrRefreshInvalid)
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	ctx := context.Background()
	users := NewUserRepository(newAccountTestQueries(t))

	user := testUser()
	require.NoError(t, users.Save(ctx, user))

	hash, err := auth.HashPassword("brand-new-password")
	require.NoError(t, err)
	require.NoError(t, users.UpdatePassword(ctx, user.ID, hash))

	got, err := users.FindByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, hash, got.PasswordHash)

	require.ErrorIs(t, users.UpdatePassword(ctx, "missing", hash), auth.ErrUserNotFound)
}

func TestUserRepository_Delete(t *testing.T) {
	ctx := context.Background()
	queries := newAccountTestQueries(t)
	users := NewUserRepository(queries)
	tokens := NewTokenRepository(queries)

	user := testUser()
	require.NoError(t, users.Save(ctx, user))
	entry := testRefreshEntry(user.ID)
	require.NoError(t, tokens.SaveRefresh(ctx, entry))

	require.NoError(t, users.Delete(ctx, user.ID))

	_, err := users.FindByID(ctx, user.ID)
	require.ErrorIs(t, err, auth.ErrUserNotFound)
	_, err = tokens.FindRefresh(ctx, entry.Token)
	require.ErrorIs(t, err, auth.ErrRefreshInvalid)

	require.ErrorIs(t, users.Delete(ctx, user.ID), auth.ErrUserNotFound)
}

func TestOutboxRepository_UserDeletedLifecycle(t *testing.T) {
	ctx := context.Background()
	outbox := NewOutboxRepository(newAccountTestQueries(t))

	user := testUser()
	user.ID = auth.UserID(uuid.NewString())
	deletedAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, outbox.SaveUserDeleted(ctx, user.ID, deletedAt))

	pending, err := outbox.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, contract.SubjectDeleted, pending[0].Subject)
	require.Equal(t, contract.SubjectDeleted+":"+string(user.ID), pending[0].ID)

	var event contract.DeletedEvent
	require.NoError(t, json.Unmarshal(pending[0].Payload, &event))
	require.Equal(t, string(user.ID), event.UserID)
	require.True(t, deletedAt.Equal(event.DeletedAt))

	require.NoError(t, outbox.MarkPublished(ctx, pending[0].ID))

	pending, err = outbox.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, pending)
}
