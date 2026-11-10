package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func newTestBlock(t *testing.T, userID, blockedID user.ID) block.Block {
	t.Helper()

	b, err := block.New(userID, blockedID, testCreatedAt)
	require.NoError(t, err)
	return b
}

func TestBlockRepository(t *testing.T) {
	ctx := context.Background()
	users, contacts := newTestRepositories(t)
	blocks := NewBlockRepository(users.q)

	saveTestUser(t, ctx, users, "owner", "owner@example.com")
	target := saveTestUser(t, ctx, users, "target", "target@example.com")
	require.NoError(t, contacts.Add(ctx, newTestContact(t, "owner", "target")))

	require.NoError(t, blocks.Add(ctx, newTestBlock(t, "owner", "target")))
	require.ErrorIs(t, blocks.Add(ctx, newTestBlock(t, "owner", "target")), block.ErrAlreadyBlocked)
	require.ErrorIs(t, blocks.Add(ctx, newTestBlock(t, "owner", "missing")), user.ErrNotFound)

	for _, id := range []user.ID{"owner", "target"} {
		got, err := contacts.GetByUserID(ctx, id, outbound.Page{Limit: 10})
		require.NoError(t, err)
		require.Empty(t, got)
	}

	exists, err := blocks.ExistsBetween(ctx, "target", "owner")
	require.NoError(t, err)
	require.True(t, exists)

	blocked, err := blocks.ListBlocked(ctx, "owner", outbound.Page{Limit: 10})
	require.NoError(t, err)
	require.Len(t, blocked, 1)
	requireSameUser(t, target, blocked[0])

	require.NoError(t, blocks.Remove(ctx, "owner", "target"))
	require.ErrorIs(t, blocks.Remove(ctx, "owner", "target"), block.ErrNotFound)

	exists, err = blocks.ExistsBetween(ctx, "owner", "target")
	require.NoError(t, err)
	require.False(t, exists)
}
