package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
)

func TestCreateChatUseCase_Execute(t *testing.T) {
	alice, bob := newUserID(), newUserID()
	existing := newPrivateChat(t, alice, bob)
	anyChat := mock.AnythingOfType("chat.Chat")

	tests := []struct {
		name    string
		u2      chat.UserID
		setup   func(*MockChatRepository, *MockBlockChecker)
		wantID  chat.ID
		wantErr error
	}{
		{
			name: "creates chat",
			u2:   bob,
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker) {
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, nil).Once()
				repo.EXPECT().Save(mock.Anything, anyChat).Return(nil).Once()
			},
		},
		{
			name: "returns existing chat",
			u2:   bob,
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker) {
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, nil).Once()
				repo.EXPECT().Save(mock.Anything, anyChat).Return(ErrChatAlreadyExists).Once()
				repo.EXPECT().FindPrivateByParticipants(mock.Anything, alice, bob).Return(existing, nil).Once()
			},
			wantID: existing.ID(),
		},
		{
			name: "blocked users",
			u2:   bob,
			setup: func(_ *MockChatRepository, blocks *MockBlockChecker) {
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(true, nil).Once()
			},
			wantErr: ErrBlocked,
		},
		{
			name: "block check fails",
			u2:   bob,
			setup: func(_ *MockChatRepository, blocks *MockBlockChecker) {
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, errTest).Once()
			},
			wantErr: errTest,
		},
		{
			name:    "chat with self",
			u2:      alice,
			setup:   func(_ *MockChatRepository, _ *MockBlockChecker) {},
			wantErr: chat.ErrInvalidParticipants,
		},
		{
			name: "save fails",
			u2:   bob,
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker) {
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, nil).Once()
				repo.EXPECT().Save(mock.Anything, anyChat).Return(ErrPersistence).Once()
			},
			wantErr: ErrPersistence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockChatRepository(t)
			blocks := NewMockBlockChecker(t)
			tt.setup(repo, blocks)

			got, err := NewCreateChat(repo, blocks).Execute(context.Background(), alice, tt.u2)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.True(t, got.HasParticipant(alice))
			require.True(t, got.HasParticipant(bob))
			if tt.wantID != "" {
				require.Equal(t, tt.wantID, got.ID())
			}
		})
	}
}

func TestGetChatUseCase_Execute(t *testing.T) {
	alice, bob := newUserID(), newUserID()
	c := newPrivateChat(t, alice, bob)

	t.Run("participant gets chat", func(t *testing.T) {
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, c.ID()).Return(c, nil).Once()

		got, err := NewGetChat(repo).Execute(context.Background(), c.ID(), alice)
		require.NoError(t, err)
		require.Equal(t, c.ID(), got.ID())
	})

	t.Run("outsider is forbidden", func(t *testing.T) {
		repo := NewMockChatRepository(t)
		repo.EXPECT().FindByID(mock.Anything, c.ID()).Return(c, nil).Once()

		_, err := NewGetChat(repo).Execute(context.Background(), c.ID(), newUserID())
		require.ErrorIs(t, err, ErrForbidden)
	})

	t.Run("invalid IDs", func(t *testing.T) {
		uc := NewGetChat(NewMockChatRepository(t))

		_, err := uc.Execute(context.Background(), "bad", alice)
		require.ErrorIs(t, err, chat.ErrInvalidChatID)
		_, err = uc.Execute(context.Background(), c.ID(), "bad")
		require.ErrorIs(t, err, chat.ErrInvalidUserID)
	})
}

func TestListChatsUseCase_Execute(t *testing.T) {
	owner := newUserID()
	chats := []chat.Chat{
		newPrivateChat(t, owner, newUserID()),
		newPrivateChat(t, owner, newUserID()),
		newPrivateChat(t, owner, newUserID()),
	}
	before := &ChatCursor{CreatedAt: time.Now(), ID: chat.NewID()}

	t.Run("returns next cursor when more chats exist", func(t *testing.T) {
		repo := NewMockChatLister(t)
		repo.EXPECT().ListByUser(mock.Anything, owner, ChatPage{Limit: 3, Before: before}).Return(chats, nil).Once()

		got, err := NewListChats(repo).Execute(context.Background(), owner, 2, before)
		require.NoError(t, err)
		require.Len(t, got.Chats, 2)
		require.NotNil(t, got.Next)
		require.Equal(t, chats[1].ID(), got.Next.ID)
		require.Equal(t, chats[1].CreatedAt(), got.Next.CreatedAt)
	})

	t.Run("last page has no cursor", func(t *testing.T) {
		repo := NewMockChatLister(t)
		repo.EXPECT().ListByUser(mock.Anything, owner, ChatPage{Limit: 4}).Return(chats, nil).Once()

		got, err := NewListChats(repo).Execute(context.Background(), owner, 3, nil)
		require.NoError(t, err)
		require.Len(t, got.Chats, 3)
		require.Nil(t, got.Next)
	})

	t.Run("invalid limit uses default", func(t *testing.T) {
		repo := NewMockChatLister(t)
		repo.EXPECT().ListByUser(mock.Anything, owner, ChatPage{Limit: DefaultChatsLimit + 1}).Return(nil, nil).Twice()

		uc := NewListChats(repo)
		_, err := uc.Execute(context.Background(), owner, 0, nil)
		require.NoError(t, err)
		_, err = uc.Execute(context.Background(), owner, MaxChatsLimit+1, nil)
		require.NoError(t, err)
	})

	t.Run("invalid user", func(t *testing.T) {
		_, err := NewListChats(NewMockChatLister(t)).Execute(context.Background(), "bad", 0, nil)
		require.ErrorIs(t, err, chat.ErrInvalidUserID)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := NewMockChatLister(t)
		repo.EXPECT().ListByUser(mock.Anything, owner, mock.Anything).Return(nil, ErrPersistence).Once()

		_, err := NewListChats(repo).Execute(context.Background(), owner, 0, nil)
		require.ErrorIs(t, err, ErrPersistence)
	})
}

func TestSendMessageUseCase_Execute(t *testing.T) {
	alice, bob, carol := newUserID(), newUserID(), newUserID()
	private := newPrivateChat(t, alice, bob)
	group := newGroupChat(t, alice, bob)

	input := func(chatID chat.ID, from chat.UserID) SendMessageInput {
		return SendMessageInput{
			MessageID: uuid.NewString(),
			ChatID:    chatID,
			FromUser:  from,
			Type:      message.TypeText,
			Payload:   json.RawMessage(`{"text":"hi"}`),
		}
	}

	tests := []struct {
		name    string
		input   SendMessageInput
		setup   func(*MockChatRepository, *MockBlockChecker, *MockMessagePublisher)
		wantErr error
	}{
		{
			name:  "publishes in private chat",
			input: input(private.ID(), alice),
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker, pub *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, private.ID()).Return(private, nil).Once()
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, nil).Once()
				pub.EXPECT().Publish(mock.Anything, mock.AnythingOfType("message.SendMessageCommand")).Return(nil).Once()
			},
		},
		{
			name:  "group messages skip block checks",
			input: input(group.ID(), bob),
			setup: func(repo *MockChatRepository, _ *MockBlockChecker, pub *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, group.ID()).Return(group, nil).Once()
				pub.EXPECT().Publish(mock.Anything, mock.AnythingOfType("message.SendMessageCommand")).Return(nil).Once()
			},
		},
		{
			name:  "blocked in private chat",
			input: input(private.ID(), bob),
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker, _ *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, private.ID()).Return(private, nil).Once()
				blocks.EXPECT().ExistsBetween(mock.Anything, bob, alice).Return(true, nil).Once()
			},
			wantErr: ErrBlocked,
		},
		{
			name:  "outsider",
			input: input(private.ID(), carol),
			setup: func(repo *MockChatRepository, _ *MockBlockChecker, _ *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, private.ID()).Return(private, nil).Once()
			},
			wantErr: ErrForbidden,
		},
		{
			name:  "chat not found",
			input: input(private.ID(), alice),
			setup: func(repo *MockChatRepository, _ *MockBlockChecker, _ *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, private.ID()).Return(chat.Chat{}, ErrChatNotFound).Once()
			},
			wantErr: ErrChatNotFound,
		},
		{
			name:    "invalid command",
			input:   SendMessageInput{MessageID: "bad", ChatID: private.ID(), FromUser: alice, Type: message.TypeText},
			setup:   func(_ *MockChatRepository, _ *MockBlockChecker, _ *MockMessagePublisher) {},
			wantErr: message.ErrInvalidMessageID,
		},
		{
			name:  "publish fails",
			input: input(private.ID(), alice),
			setup: func(repo *MockChatRepository, blocks *MockBlockChecker, pub *MockMessagePublisher) {
				repo.EXPECT().FindByID(mock.Anything, private.ID()).Return(private, nil).Once()
				blocks.EXPECT().ExistsBetween(mock.Anything, alice, bob).Return(false, nil).Once()
				pub.EXPECT().Publish(mock.Anything, mock.AnythingOfType("message.SendMessageCommand")).Return(ErrPublishFailed).Once()
			},
			wantErr: ErrPublishFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockChatRepository(t)
			blocks := NewMockBlockChecker(t)
			pub := NewMockMessagePublisher(t)
			tt.setup(repo, blocks, pub)

			id, err := NewSendMessage(repo, blocks, pub).Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.input.MessageID, id)
		})
	}
}

func TestApplyBlockUseCase_Execute(t *testing.T) {
	change := BlockChange{UserID: newUserID(), BlockedID: newUserID(), Blocked: true, At: time.Now()}

	t.Run("applies change", func(t *testing.T) {
		blocks := NewMockBlockRepository(t)
		blocks.EXPECT().Apply(mock.Anything, change).Return(nil).Once()
		require.NoError(t, NewApplyBlock(blocks).Execute(context.Background(), change))
	})

	t.Run("rejects invalid IDs", func(t *testing.T) {
		uc := NewApplyBlock(NewMockBlockRepository(t))
		require.ErrorIs(t, uc.Execute(context.Background(), BlockChange{UserID: "bad", BlockedID: change.BlockedID}), chat.ErrInvalidUserID)
		require.ErrorIs(t, uc.Execute(context.Background(), BlockChange{UserID: change.UserID, BlockedID: "bad"}), chat.ErrInvalidUserID)
	})
}
