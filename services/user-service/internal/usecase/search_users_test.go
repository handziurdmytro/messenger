package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSearchUsersUseCase_Execute(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	u, err := user.New("user-1", "test@gmail.com", now)
	require.NoError(t, err)

	tests := []struct {
		name       string
		query      string
		limit      int
		want       []user.User
		wantErr    error
		setupMocks func(*MockUserRepository)
	}{
		{
			name:  "success",
			query: "test",
			limit: 10,
			want:  []user.User{u},
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					Search(mock.Anything, "test", 10).
					Return([]user.User{u}, nil).
					Once()
			},
		},
		{
			name:  "no matches",
			query: "missing",
			limit: 10,
			want:  []user.User{},
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					Search(mock.Anything, "missing", 10).
					Return([]user.User{}, nil).
					Once()
			},
		},
		{
			name:       "empty query",
			query:      "",
			limit:      10,
			setupMocks: func(_ *MockUserRepository) {},
		},
		{
			name:  "zero limit uses default",
			query: "test",
			limit: 0,
			want:  []user.User{u},
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					Search(mock.Anything, "test", 20).
					Return([]user.User{u}, nil).
					Once()
			},
		},
		{
			name:  "too large limit uses default",
			query: "test",
			limit: 51,
			want:  []user.User{u},
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					Search(mock.Anything, "test", 20).
					Return([]user.User{u}, nil).
					Once()
			},
		},
		{
			name:    "repository error",
			query:   "test",
			limit:   10,
			wantErr: outbound.ErrPersistence,
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					Search(mock.Anything, "test", 10).
					Return(nil, outbound.ErrPersistence).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tt.setupMocks(users)

			uc := NewSearchUsersUseCase(users)
			got, err := uc.Execute(context.Background(), tt.query, tt.limit)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
