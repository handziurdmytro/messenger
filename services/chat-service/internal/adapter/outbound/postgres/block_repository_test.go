package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
)

func TestBlockRepository(t *testing.T) {
	_, pool := newTestChatRepository(t)
	repo := NewBlockRepository(pool)
	ctx := context.Background()

	owner := chat.UserID(uuid.NewString())
	target := chat.UserID(uuid.NewString())
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	exists, err := repo.ExistsBetween(ctx, owner, target)
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, repo.Apply(ctx, usecase.BlockChange{UserID: owner, BlockedID: target, Blocked: true, At: at}))

	exists, err = repo.ExistsBetween(ctx, target, owner)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, repo.Apply(ctx, usecase.BlockChange{UserID: owner, BlockedID: target, Blocked: false, At: at.Add(-time.Minute)}))

	exists, err = repo.ExistsBetween(ctx, owner, target)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, repo.Apply(ctx, usecase.BlockChange{UserID: owner, BlockedID: target, Blocked: false, At: at.Add(time.Minute)}))

	exists, err = repo.ExistsBetween(ctx, owner, target)
	require.NoError(t, err)
	require.False(t, exists)
}
