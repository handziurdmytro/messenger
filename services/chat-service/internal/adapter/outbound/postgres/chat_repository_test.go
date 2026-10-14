package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/pgkit/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newTestChatRepository(t *testing.T) (*ChatRepository, *pgxpool.Pool) {
	t.Helper()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")
	return NewChatRepository(pg.Pool), pg.Pool
}

func newTestPrivateChat(t *testing.T) chat.Chat {
	t.Helper()

	c, err := chat.NewPrivate(chat.UserID(uuid.NewString()), chat.UserID(uuid.NewString()))
	require.NoError(t, err)
	return c
}

func requireSameChat(t *testing.T, expected, actual chat.Chat) {
	t.Helper()

	require.Equal(t, expected.ID(), actual.ID())
	require.Equal(t, expected.Type(), actual.Type())
	require.ElementsMatch(t, expected.Participants(), actual.Participants())
	require.WithinDuration(t, expected.CreatedAt(), actual.CreatedAt(), time.Microsecond)
}

func TestChatRepository_Save(t *testing.T) {
	ctx := context.Background()
	repo, pool := newTestChatRepository(t)

	t.Run("creates chat and participants", func(t *testing.T) {
		c := newTestPrivateChat(t)
		require.NoError(t, repo.Save(ctx, c))

		found, err := repo.FindByID(ctx, c.ID())
		require.NoError(t, err)
		requireSameChat(t, c, found)

		var count int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM participants WHERE chat_id = $1", string(c.ID())).Scan(&count))
		require.Equal(t, 2, count)
	})

	t.Run("same participants in reverse order", func(t *testing.T) {
		original := newTestPrivateChat(t)
		require.NoError(t, repo.Save(ctx, original))

		users := original.Participants()
		duplicate, err := chat.NewPrivate(users[1], users[0])
		require.NoError(t, err)

		require.ErrorIs(t, repo.Save(ctx, duplicate), usecase.ErrChatAlreadyExists)

		found, err := repo.FindPrivateByParticipants(ctx, users[0], users[1])
		require.NoError(t, err)
		requireSameChat(t, original, found)

		_, err = repo.FindByID(ctx, duplicate.ID())
		require.ErrorIs(t, err, usecase.ErrChatNotFound)
	})

	t.Run("same chat id", func(t *testing.T) {
		c := newTestPrivateChat(t)
		require.NoError(t, repo.Save(ctx, c))
		require.ErrorIs(t, repo.Save(ctx, c), usecase.ErrPersistence)
	})

	t.Run("rolls back if participant insert fails", func(t *testing.T) {
		c := newTestPrivateChat(t)
		blocked := c.Participants()[0]
		constraint := fmt.Sprintf("ALTER TABLE participants ADD CONSTRAINT chk_test_blocked_user CHECK (user_id <> '%s'::uuid)", blocked)
		_, err := pool.Exec(ctx, constraint)
		require.NoError(t, err)

		err = repo.Save(ctx, c)
		require.ErrorIs(t, err, usecase.ErrPersistence)

		found, err := repo.FindByID(ctx, c.ID())
		require.ErrorIs(t, err, usecase.ErrChatNotFound)
		require.Equal(t, chat.Chat{}, found)
	})
}

func TestChatRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newTestChatRepository(t)

	t.Run("found", func(t *testing.T) {
		c := newTestPrivateChat(t)
		require.NoError(t, repo.Save(ctx, c))

		found, err := repo.FindByID(ctx, c.ID())
		require.NoError(t, err)
		requireSameChat(t, c, found)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.FindByID(ctx, chat.ID(uuid.NewString()))
		require.ErrorIs(t, err, usecase.ErrChatNotFound)
		require.Equal(t, chat.Chat{}, found)
	})
}

func TestChatRepository_FindPrivateByParticipants(t *testing.T) {
	ctx := context.Background()
	repo, _ := newTestChatRepository(t)

	c := newTestPrivateChat(t)
	require.NoError(t, repo.Save(ctx, c))
	users := c.Participants()

	t.Run("found", func(t *testing.T) {
		found, err := repo.FindPrivateByParticipants(ctx, users[0], users[1])
		require.NoError(t, err)
		requireSameChat(t, c, found)
	})

	t.Run("reverse order", func(t *testing.T) {
		found, err := repo.FindPrivateByParticipants(ctx, users[1], users[0])
		require.NoError(t, err)
		requireSameChat(t, c, found)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.FindPrivateByParticipants(ctx, users[0], chat.UserID(uuid.NewString()))
		require.ErrorIs(t, err, usecase.ErrChatNotFound)
		require.Equal(t, chat.Chat{}, found)
	})
}
