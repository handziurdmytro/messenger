package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func TestDeleteUserUseCase_Execute(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := createdAt.Add(time.Hour)

	active, err := user.New("user-1", "alice@example.com", createdAt)
	require.NoError(t, err)
	alreadyDeleted := active
	require.NoError(t, alreadyDeleted.Delete(deletedAt))

	anonymized := mock.MatchedBy(func(u user.User) bool {
		return u.ID() == "user-1" && u.IsDeleted() && u.DeletedAt().Equal(deletedAt) && u.Email().String() == ""
	})

	tests := []struct {
		name      string
		deletedAt time.Time
		setup     func(*MockUserRepository)
		wantErr   error
	}{
		{
			name:      "marks user deleted",
			deletedAt: deletedAt,
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-1")).Return(active, nil).Once()
				users.EXPECT().MarkDeleted(mock.Anything, anonymized).Return(nil).Once()
			},
		},
		{
			name:      "already deleted is a no-op",
			deletedAt: deletedAt,
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-1")).Return(alreadyDeleted, nil).Once()
			},
		},
		{
			name:      "user not found",
			deletedAt: deletedAt,
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-1")).Return(user.User{}, user.ErrNotFound).Once()
			},
			wantErr: user.ErrNotFound,
		},
		{
			name:      "deleted before creation",
			deletedAt: createdAt.Add(-time.Hour),
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-1")).Return(active, nil).Once()
			},
			wantErr: user.ErrInvalidTimestamp,
		},
		{
			name:      "repository error",
			deletedAt: deletedAt,
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID("user-1")).Return(active, nil).Once()
				users.EXPECT().MarkDeleted(mock.Anything, anonymized).Return(outbound.ErrPersistence).Once()
			},
			wantErr: outbound.ErrPersistence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tt.setup(users)

			err := NewDeleteUserUseCase(users).Execute(context.Background(), "user-1", tt.deletedAt)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
