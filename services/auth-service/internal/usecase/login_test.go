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

func TestLoginUseCase_Execute(t *testing.T) {
	const (
		email    = "valid@gmail.com"
		password = "pass1234"
	)

	tokenSvc, err := auth.NewTokenService(
		generateTestRSAKeyPEM(t),
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	user, err := auth.NewUser(email, password)
	require.NoError(t, err)

	tests := []struct {
		name       string
		input      LoginInput
		setupMocks func(*MockUserRepository, *MockTokenRepository)
		wantErr    error
	}{
		{
			name: "user not found",
			input: LoginInput{
				Email:    "unknown@gmail.com",
				Password: password,
			},
			setupMocks: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, "unknown@gmail.com").
					Return(auth.User{}, auth.ErrUserNotFound)
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name: "repository failure",
			input: LoginInput{
				Email:    email,
				Password: password,
			},
			setupMocks: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(auth.User{}, errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name: "invalid password",
			input: LoginInput{
				Email:    email,
				Password: "wrong-password",
			},
			setupMocks: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(user, nil)
			},
			wantErr: auth.ErrPasswordInvalid,
		},
		{
			name: "save refresh failed",
			input: LoginInput{
				Email:    email,
				Password: password,
			},
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(user, nil)

				tokens.EXPECT().
					SaveRefresh(
						mock.Anything,
						mock.MatchedBy(func(entry auth.RefreshEntry) bool {
							return entry.UserID == user.ID
						}),
					).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name: "success",
			input: LoginInput{
				Email:    email,
				Password: password,
			},
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(user, nil)

				tokens.EXPECT().
					SaveRefresh(
						mock.Anything,
						mock.MatchedBy(func(entry auth.RefreshEntry) bool {
							return entry.UserID == user.ID &&
								entry.Token != ""
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

			uc := NewLoginUseCase(users, tokens, tokenSvc)

			output, err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, LoginOutput{}, output)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, user.ID, output.UserID)
			assert.NotEmpty(t, output.Tokens.Access)
			assert.NotEmpty(t, output.Tokens.Refresh)
			assert.False(t, output.Tokens.ExpiresAt.IsZero())
		})
	}
}
