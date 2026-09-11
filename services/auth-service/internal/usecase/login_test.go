package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoginUseCase_Execute(t *testing.T) {
	const (
		email         = "valid@gmail.com"
		validPassword = "pass1234"
	)

	testPEM := generateTestRSAKeyPEM(t)

	tokenService, err := auth.NewTokenService(
		testPEM,
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	testUser, err := auth.NewUser(email, validPassword)
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
				Password: validPassword,
			},
			setupMocks: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, "unknown@gmail.com").
					Return(auth.User{}, auth.ErrUserNotFound)
			},
			wantErr: auth.ErrUserNotFound,
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
					Return(testUser, nil)
			},
			wantErr: auth.ErrPasswordInvalid,
		},
		{
			name: "refresh token persistence failed",
			input: LoginInput{
				Email:    email,
				Password: validPassword,
			},
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(testUser, nil)

				tokens.EXPECT().
					SaveRefresh(mock.Anything, mock.Anything).
					Return(errors.New("database connection lost"))
			},
			wantErr: ErrInternal,
		},
		{
			name: "success",
			input: LoginInput{
				Email:    email,
				Password: validPassword,
			},
			setupMocks: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().
					FindByEmail(mock.Anything, email).
					Return(testUser, nil)

				tokens.EXPECT().
					SaveRefresh(
						mock.Anything,
						mock.MatchedBy(func(entry auth.RefreshEntry) bool {
							return entry.UserID == testUser.ID
						}),
					).
					Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tokens := NewMockTokenRepository(t)

			tt.setupMocks(users, tokens)

			uc := NewLoginUseCase(users, tokens, tokenService)

			_, err := uc.Execute(context.Background(), tt.input)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
