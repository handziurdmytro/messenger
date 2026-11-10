package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func TestBlockUserUseCase_Execute(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	target, err := user.New("user-2", "target@example.com", createdAt)
	require.NoError(t, err)
	deleted := target
	require.NoError(t, deleted.Delete(createdAt.Add(time.Hour)))

	validBlock := mock.MatchedBy(func(b block.Block) bool {
		return b.UserID() == "user-1" && b.BlockedID() == "user-2" && !b.CreatedAt().IsZero()
	})

	tests := []struct {
		name      string
		blockedID user.ID
		setup     func(*MockUserRepository, *MockBlockRepository)
		wantErr   error
	}{
		{
			name:      "blocks user",
			blockedID: "user-2",
			setup: func(users *MockUserRepository, blocks *MockBlockRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-2")).Return(target, nil).Once()
				blocks.EXPECT().Add(mock.Anything, validBlock).Return(nil).Once()
			},
		},
		{
			name:      "cannot block self",
			blockedID: "user-1",
			setup:     func(_ *MockUserRepository, _ *MockBlockRepository) {},
			wantErr:   block.ErrCannotBlockSelf,
		},
		{
			name:      "target not found",
			blockedID: "user-2",
			setup: func(users *MockUserRepository, _ *MockBlockRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-2")).Return(user.User{}, user.ErrNotFound).Once()
			},
			wantErr: user.ErrNotFound,
		},
		{
			name:      "target deleted",
			blockedID: "user-2",
			setup: func(users *MockUserRepository, _ *MockBlockRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-2")).Return(deleted, nil).Once()
			},
			wantErr: user.ErrNotFound,
		},
		{
			name:      "already blocked",
			blockedID: "user-2",
			setup: func(users *MockUserRepository, blocks *MockBlockRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-2")).Return(target, nil).Once()
				blocks.EXPECT().Add(mock.Anything, validBlock).Return(block.ErrAlreadyBlocked).Once()
			},
			wantErr: block.ErrAlreadyBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			blocks := NewMockBlockRepository(t)
			tt.setup(users, blocks)

			err := NewBlockUserUseCase(users, blocks).Execute(context.Background(), "user-1", tt.blockedID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUnblockUserUseCase_Execute(t *testing.T) {
	t.Run("removes block", func(t *testing.T) {
		blocks := NewMockBlockRepository(t)
		blocks.EXPECT().Remove(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(nil).Once()

		require.NoError(t, NewUnblockUserUseCase(blocks).Execute(context.Background(), "user-1", "user-2"))
	})

	t.Run("validates IDs", func(t *testing.T) {
		uc := NewUnblockUserUseCase(NewMockBlockRepository(t))
		require.ErrorIs(t, uc.Execute(context.Background(), "", "user-2"), block.ErrUserIDEmpty)
		require.ErrorIs(t, uc.Execute(context.Background(), "user-1", ""), block.ErrBlockedIDEmpty)
	})

	t.Run("not found", func(t *testing.T) {
		blocks := NewMockBlockRepository(t)
		blocks.EXPECT().Remove(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(block.ErrNotFound).Once()

		err := NewUnblockUserUseCase(blocks).Execute(context.Background(), "user-1", "user-2")
		require.ErrorIs(t, err, block.ErrNotFound)
	})
}

func TestListBlockedUseCase_Execute(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		offset   int
		wantPage outbound.Page
	}{
		{"default limit", 0, 0, outbound.Page{Limit: defaultBlocksLimit}},
		{"too large limit", maxBlocksLimit + 1, 5, outbound.Page{Limit: defaultBlocksLimit, Offset: 5}},
		{"negative offset", 10, -3, outbound.Page{Limit: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := NewMockBlockRepository(t)
			blocks.EXPECT().ListBlocked(mock.Anything, user.ID("user-1"), tt.wantPage).Return([]user.User{}, nil).Once()

			got, err := NewListBlockedUseCase(blocks).Execute(context.Background(), "user-1", tt.limit, tt.offset)
			require.NoError(t, err)
			require.Empty(t, got)
		})
	}

	t.Run("repository error", func(t *testing.T) {
		blocks := NewMockBlockRepository(t)
		blocks.EXPECT().ListBlocked(mock.Anything, user.ID("user-1"), mock.Anything).Return(nil, outbound.ErrPersistence).Once()

		_, err := NewListBlockedUseCase(blocks).Execute(context.Background(), "user-1", 0, 0)
		require.ErrorIs(t, err, outbound.ErrPersistence)
	})
}
