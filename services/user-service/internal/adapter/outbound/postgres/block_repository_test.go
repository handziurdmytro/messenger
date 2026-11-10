package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
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
	outbox := NewOutboxRepository(users.q)

	owner := user.ID(uuid.NewString())
	targetID := user.ID(uuid.NewString())
	saveTestUser(t, ctx, users, owner, "owner@example.com")
	target := saveTestUser(t, ctx, users, targetID, "target@example.com")
	require.NoError(t, contacts.Add(ctx, newTestContact(t, owner, targetID)))

	require.NoError(t, blocks.Add(ctx, newTestBlock(t, owner, targetID)))
	require.ErrorIs(t, blocks.Add(ctx, newTestBlock(t, owner, targetID)), block.ErrAlreadyBlocked)
	require.ErrorIs(t, blocks.Add(ctx, newTestBlock(t, owner, user.ID(uuid.NewString()))), user.ErrNotFound)

	pending, err := outbox.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, contract.SubjectBlocked, pending[0].Subject)

	var event contract.BlockEvent
	require.NoError(t, json.Unmarshal(pending[0].Payload, &event))
	require.Equal(t, owner.String(), event.UserID)
	require.Equal(t, targetID.String(), event.BlockedID)

	for _, id := range []user.ID{owner, targetID} {
		got, err := contacts.GetByUserID(ctx, id, outbound.Page{Limit: 10})
		require.NoError(t, err)
		require.Empty(t, got)
	}

	exists, err := blocks.ExistsBetween(ctx, targetID, owner)
	require.NoError(t, err)
	require.True(t, exists)

	blocked, err := blocks.ListBlocked(ctx, owner, outbound.Page{Limit: 10})
	require.NoError(t, err)
	require.Len(t, blocked, 1)
	requireSameUser(t, target, blocked[0])

	require.NoError(t, blocks.Remove(ctx, owner, targetID))
	require.ErrorIs(t, blocks.Remove(ctx, owner, targetID), block.ErrNotFound)

	pending, err = outbox.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	require.Equal(t, contract.SubjectUnblocked, pending[1].Subject)

	exists, err = blocks.ExistsBetween(ctx, owner, targetID)
	require.NoError(t, err)
	require.False(t, exists)
}
