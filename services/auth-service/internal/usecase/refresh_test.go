package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRefreshUseCase_Execute(t *testing.T) {
	const (
		email    = "valid@gmail.com"
		password = "pass1234"
	)

	oldToken := auth.RefreshToken("old-refresh-token")

	tokenSvc, err := auth.NewTokenService(
		generateTestRSAKeyPEM(t),
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	user, err := auth.NewUser(email, password)
	require.NoError(t, err)

	activeEntry := auth.RefreshEntry{
		UserID:    user.ID,
		Token:     oldToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	expiredEntry := auth.RefreshEntry{
		UserID:    user.ID,
		Token:     oldToken,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	tests := []struct {
		name       string
		setupMocks func(*MockUserRepository, *MockTokenRepository)
		wantErr    error
	}{
		{
			name: "invalid refresh token",
			setupMocks: func(_ *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(auth.RefreshEntry{}, auth.ErrRefreshInvalid)
			},
			wantErr: auth.ErrRefreshInvalid,
		},
		{
			name: "expired refresh token",
			setupMocks: func(_ *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(expiredEntry, nil)

				tokens.EXPECT().
					DeleteRefresh(mock.Anything, oldToken).
					Return(nil)
			},
			wantErr: auth.ErrRefreshExpired,
		},
		{
			name: "user not found",
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(activeEntry, nil)

				users.EXPECT().
					FindByID(mock.Anything, user.ID).
					Return(auth.User{}, auth.ErrUserNotFound)
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name: "delete refresh failed",
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(activeEntry, nil)

				users.EXPECT().
					FindByID(mock.Anything, user.ID).
					Return(user, nil)

				tokens.EXPECT().
					DeleteRefresh(mock.Anything, oldToken).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name: "save refresh failed",
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(activeEntry, nil)

				users.EXPECT().
					FindByID(mock.Anything, user.ID).
					Return(user, nil)

				tokens.EXPECT().
					DeleteRefresh(mock.Anything, oldToken).
					Return(nil)

				tokens.EXPECT().
					SaveRefresh(mock.Anything, mock.Anything).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name: "success",
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				tokens.EXPECT().
					FindRefresh(mock.Anything, oldToken).
					Return(activeEntry, nil)

				users.EXPECT().
					FindByID(mock.Anything, user.ID).
					Return(user, nil)

				tokens.EXPECT().
					DeleteRefresh(mock.Anything, oldToken).
					Return(nil)

				tokens.EXPECT().
					SaveRefresh(
						mock.Anything,
						mock.MatchedBy(func(entry auth.RefreshEntry) bool {
							return entry.UserID == user.ID &&
								entry.Token != "" &&
								entry.Token != oldToken
						}),
					).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tokens := NewMockTokenRepository(t)

			tt.setupMocks(users, tokens)

			transactor := testTransactor(users, tokens, nil)

			uc := NewRefreshUseCase(users, tokens, transactor, tokenSvc)

			pair, err := uc.Execute(context.Background(), oldToken)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, auth.TokenPair{}, pair)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, pair.Access)
			assert.NotEmpty(t, pair.Refresh)
			assert.NotEqual(t, oldToken, pair.Refresh)
			assert.False(t, pair.ExpiresAt.IsZero())
		})
	}
}
