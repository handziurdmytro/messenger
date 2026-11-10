package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
)

func TestChatRepository_Group(t *testing.T) {
	repo, _ := newTestChatRepository(t)
	ctx := context.Background()

	owner := chat.UserID(uuid.NewString())
	member := chat.UserID(uuid.NewString())
	extra := chat.UserID(uuid.NewString())

	group, err := chat.NewGroup(owner, "Team", []chat.UserID{member})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, group))

	found, err := repo.FindByID(ctx, group.ID())
	require.NoError(t, err)
	require.Equal(t, chat.TypeGroup, found.Type())
	require.Equal(t, "Team", found.Title())
	require.Equal(t, owner, found.Owner())
	require.Equal(t, []chat.UserID{owner, member}, found.Participants())

	require.NoError(t, found.Rename(owner, "Renamed"))
	require.NoError(t, found.AddParticipants(owner, []chat.UserID{extra}))
	require.NoError(t, found.Leave(owner))
	require.NoError(t, repo.Update(ctx, found))

	updated, err := repo.FindByID(ctx, group.ID())
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Title())
	require.Equal(t, member, updated.Owner())
	require.ElementsMatch(t, []chat.UserID{member, extra}, updated.Participants())
	require.False(t, updated.HasParticipant(owner))

	listed, err := repo.ListByUser(ctx, extra, usecase.ChatPage{Limit: 10})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "Renamed", listed[0].Title())
	require.Equal(t, member, listed[0].Owner())

	ownerChats, err := repo.ListByUser(ctx, owner, usecase.ChatPage{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, ownerChats)
}
