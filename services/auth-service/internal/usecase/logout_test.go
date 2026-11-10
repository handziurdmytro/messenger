package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

func TestLogoutUseCase_Execute(t *testing.T) {
	const token auth.RefreshToken = "refresh-token"

	tests := []struct {
		name    string
		token   auth.RefreshToken
		setup   func(*MockTokenRepository)
		wantErr error
	}{
		{
			name:  "deletes refresh token",
			token: token,
			setup: func(tokens *MockTokenRepository) {
				tokens.EXPECT().DeleteRefresh(mock.Anything, token).Return(nil).Once()
			},
		},
		{
			name:    "empty token",
			setup:   func(_ *MockTokenRepository) {},
			wantErr: auth.ErrRefreshInvalid,
		},
		{
			name:  "repository error",
			token: token,
			setup: func(tokens *MockTokenRepository) {
				tokens.EXPECT().DeleteRefresh(mock.Anything, token).Return(outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := NewMockTokenRepository(t)
			tt.setup(tokens)

			err := NewLogoutUseCase(tokens).Execute(context.Background(), tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLogoutAllUseCase_Execute(t *testing.T) {
	const userID auth.UserID = "user-1"

	tests := []struct {
		name    string
		userID  auth.UserID
		setup   func(*MockTokenRepository)
		wantErr error
	}{
		{
			name:   "deletes every refresh token of the user",
			userID: userID,
			setup: func(tokens *MockTokenRepository) {
				tokens.EXPECT().DeleteAllRefresh(mock.Anything, userID).Return(nil).Once()
			},
		},
		{
			name:    "empty user ID",
			setup:   func(_ *MockTokenRepository) {},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:   "repository error",
			userID: userID,
			setup: func(tokens *MockTokenRepository) {
				tokens.EXPECT().DeleteAllRefresh(mock.Anything, userID).Return(outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := NewMockTokenRepository(t)
			tt.setup(tokens)

			err := NewLogoutAllUseCase(tokens).Execute(context.Background(), tt.userID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
