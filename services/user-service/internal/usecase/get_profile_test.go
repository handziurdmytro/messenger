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

func TestGetProfileUseCase_Execute(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	u, err := user.New("valid", "valid@gmail.com", now)
	require.NoError(t, err)

	tests := []struct {
		name       string
		input      user.ID
		setupMocks func(*MockUserRepository)
		want       user.User
		wantErr    error
	}{
		{
			name:  "success",
			input: "valid",
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("valid")).
					Return(u, nil).
					Once()
			},
			want: u,
		},
		{
			name:  "user not found",
			input: "invalid",
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("invalid")).
					Return(user.User{}, user.ErrNotFound).
					Once()
			},
			wantErr: user.ErrNotFound,
		},
		{
			name:  "repository unavailable",
			input: "valid",
			setupMocks: func(users *MockUserRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("valid")).
					Return(user.User{}, outbound.ErrPersistence).
					Once()
			},
			wantErr: outbound.ErrPersistence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tt.setupMocks(users)

			uc := NewGetProfileUseCase(users)
			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, user.User{}, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
