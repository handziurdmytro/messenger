package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

func TestCreateGroupUseCase_Execute(t *testing.T) {
	owner, member := newUserID(), newUserID()
	savedGroup := mock.MatchedBy(func(c chat.Chat) bool {
		return c.Type() == chat.TypeGroup && c.Owner() == owner && c.HasParticipant(member)
	})

	t.Run("creates group", func(t *testing.T) {
		repo := NewMockChatRepository(t)
		repo.EXPECT().Save(mock.Anything, savedGroup).Return(nil).Once()

		got, err := NewCreateGroup(repo).Execute(context.Background(), CreateGroupInput{Owner: owner, Title: "Team", Members: []chat.UserID{member}})
		require.NoError(t, err)
		require.Equal(t, "Team", got.Title())
	})

	t.Run("invalid title", func(t *testing.T) {
		_, err := NewCreateGroup(NewMockChatRepository(t)).Execute(context.Background(), CreateGroupInput{Owner: owner, Members: []chat.UserID{member}})
		require.ErrorIs(t, err, chat.ErrInvalidTitle)
	})

	t.Run("save fails", func(t *testing.T) {
		repo := NewMockChatRepository(t)
		repo.EXPECT().Save(mock.Anything, savedGroup).Return(ErrPersistence).Once()

		_, err := NewCreateGroup(repo).Execute(context.Background(), CreateGroupInput{Owner: owner, Title: "Team", Members: []chat.UserID{member}})
		require.ErrorIs(t, err, ErrPersistence)
	})
}

func TestUpdateGroupUseCase_Execute(t *testing.T) {
	owner, member, extra := newUserID(), newUserID(), newUserID()

	tests := []struct {
		name    string
		by      chat.UserID
		change  GroupChange
		save    bool
		saveErr error
		check   func(*testing.T, chat.Chat)
		wantErr error
	}{
		{
			name:   "owner renames",
			by:     owner,
			change: Rename("New title"),
			save:   true,
			check:  func(t *testing.T, c chat.Chat) { require.Equal(t, "New title", c.Title()) },
		},
		{
			name:   "owner adds participants",
			by:     owner,
			change: AddParticipants([]chat.UserID{extra}),
			save:   true,
			check:  func(t *testing.T, c chat.Chat) { require.True(t, c.HasParticipant(extra)) },
		},
		{
			name:   "owner removes participant",
			by:     owner,
			change: RemoveParticipant(member),
			save:   true,
			check:  func(t *testing.T, c chat.Chat) { require.False(t, c.HasParticipant(member)) },
		},
		{
			name:    "member cannot rename",
			by:      member,
			change:  Rename("Hack"),
			wantErr: chat.ErrNotOwner,
		},
		{
			name:    "outsider is forbidden",
			by:      extra,
			change:  Rename("Hack"),
			wantErr: ErrForbidden,
		},
		{
			name:    "update fails",
			by:      owner,
			change:  Rename("New title"),
			save:    true,
			saveErr: ErrPersistence,
			wantErr: ErrPersistence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := newGroupChat(t, owner, member)
			repo := NewMockChatRepository(t)
			repo.EXPECT().FindByID(mock.Anything, group.ID()).Return(group, nil).Once()
			if tt.save {
				repo.EXPECT().Update(mock.Anything, mock.AnythingOfType("chat.Chat")).Return(tt.saveErr).Once()
			}

			got, err := NewUpdateGroup(repo).Execute(context.Background(), group.ID(), tt.by, tt.change)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, got)
		})
	}

	t.Run("invalid user", func(t *testing.T) {
		_, err := NewUpdateGroup(NewMockChatRepository(t)).Execute(context.Background(), chat.NewID(), "bad", Rename("x"))
		require.ErrorIs(t, err, chat.ErrInvalidUserID)
	})

	t.Run("chat not found", func(t *testing.T) {
		id := chat.NewID()
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, id).Return(chat.Chat{}, ErrChatNotFound).Once()

		_, err := NewUpdateGroup(repo).Execute(context.Background(), id, owner, Rename("x"))
		require.ErrorIs(t, err, ErrChatNotFound)
	})
}

func TestDeleteChatUseCase_Execute(t *testing.T) {
	alice, bob, carol := newUserID(), newUserID(), newUserID()

	t.Run("deletes private chat", func(t *testing.T) {
		c := newPrivateChat(t, alice, bob)
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, c.ID()).Return(c, nil).Once()
		repo.EXPECT().Delete(mock.Anything, c.ID()).Return(nil).Once()

		require.NoError(t, NewDeleteChat(repo).Execute(context.Background(), c.ID(), alice))
	})

	t.Run("leaving group transfers ownership", func(t *testing.T) {
		g := newGroupChat(t, alice, bob, carol)
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, g.ID()).Return(g, nil).Once()
		repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c chat.Chat) bool {
			return !c.HasParticipant(alice) && c.Owner() == bob
		})).Return(nil).Once()

		require.NoError(t, NewDeleteChat(repo).Execute(context.Background(), g.ID(), alice))
	})

	t.Run("last member leaving deletes group", func(t *testing.T) {
		g := newGroupChat(t, alice, bob)
		require.NoError(t, g.Leave(bob))
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, g.ID()).Return(g, nil).Once()
		repo.EXPECT().Delete(mock.Anything, g.ID()).Return(nil).Once()

		require.NoError(t, NewDeleteChat(repo).Execute(context.Background(), g.ID(), alice))
	})

	t.Run("outsider is forbidden", func(t *testing.T) {
		c := newPrivateChat(t, alice, bob)
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, c.ID()).Return(c, nil).Once()

		require.ErrorIs(t, NewDeleteChat(repo).Execute(context.Background(), c.ID(), carol), ErrForbidden)
	})

	t.Run("invalid IDs", func(t *testing.T) {
		uc := NewDeleteChat(NewMockChatRepository(t))
		require.ErrorIs(t, uc.Execute(context.Background(), "bad", alice), chat.ErrInvalidChatID)
		require.ErrorIs(t, uc.Execute(context.Background(), chat.NewID(), "bad"), chat.ErrInvalidUserID)
	})
}
