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

func TestRegisterUseCase_Execute(t *testing.T) {
	const (
		email    = "test@gmail.com"
		password = "pass1234"
	)

	tokenSvc, err := auth.NewTokenService(
		generateTestRSAKeyPEM(t),
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	validInput := RegisterInput{
		Email:    email,
		Password: password,
	}

	tests := []struct {
		name       string
		input      RegisterInput
		setupMocks func(
			*MockUserRepository,
			*MockTokenRepository,
			*MockOutboxRepository,
		)
		wantErr error
	}{
		{
			name: "invalid input",
			input: RegisterInput{
				Email:    email,
				Password: "123",
			},
			wantErr: ErrInternal,
		},
		{
			name:  "email already taken",
			input: validInput,
			setupMocks: func(
				users *MockUserRepository,
				_ *MockTokenRepository,
				_ *MockOutboxRepository,
			) {
				users.EXPECT().
					Save(mock.Anything, mock.Anything).
					Return(auth.ErrEmailTaken)
			},
			wantErr: auth.ErrEmailTaken,
		},
		{
			name:  "save user failed",
			input: validInput,
			setupMocks: func(
				users *MockUserRepository,
				_ *MockTokenRepository,
				_ *MockOutboxRepository,
			) {
				users.EXPECT().
					Save(mock.Anything, mock.Anything).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name:  "save refresh failed",
			input: validInput,
			setupMocks: func(
				users *MockUserRepository,
				tokens *MockTokenRepository,
				_ *MockOutboxRepository,
			) {
				users.EXPECT().
					Save(mock.Anything, mock.Anything).
					Return(nil)

				tokens.EXPECT().
					SaveRefresh(mock.Anything, mock.Anything).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name:  "save outbox failed",
			input: validInput,
			setupMocks: func(
				users *MockUserRepository,
				tokens *MockTokenRepository,
				outbox *MockOutboxRepository,
			) {
				users.EXPECT().
					Save(mock.Anything, mock.Anything).
					Return(nil)

				tokens.EXPECT().
					SaveRefresh(mock.Anything, mock.Anything).
					Return(nil)

				outbox.EXPECT().
					SaveUserCreated(mock.Anything, mock.Anything).
					Return(errors.New("database unavailable"))
			},
			wantErr: ErrInternal,
		},
		{
			name:  "success",
			input: validInput,
			setupMocks: func(
				users *MockUserRepository,
				tokens *MockTokenRepository,
				outbox *MockOutboxRepository,
			) {
				users.EXPECT().
					Save(
						mock.Anything,
						mock.MatchedBy(func(user auth.User) bool {
							return user.ID != "" &&
								user.Email == email &&
								auth.CheckPassword(password, user.PasswordHash) == nil
						}),
					).
					Return(nil)

				tokens.EXPECT().
					SaveRefresh(
						mock.Anything,
						mock.MatchedBy(func(entry auth.RefreshEntry) bool {
							return entry.UserID != "" &&
								entry.Token != ""
						}),
					).
					Return(nil)

				outbox.EXPECT().
					SaveUserCreated(
						mock.Anything,
						mock.MatchedBy(func(user auth.User) bool {
							return user.ID != "" &&
								user.Email == email
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
			outbox := NewMockOutboxRepository(t)

			if tt.setupMocks != nil {
				tt.setupMocks(users, tokens, outbox)
			}

			transactor := testTransactor(users, tokens, outbox)

			uc := NewRegisterUseCase(transactor, tokenSvc)

			output, err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, RegisterOutput{}, output)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, output.UserID)
			assert.NotEmpty(t, output.Tokens.Access)
			assert.NotEmpty(t, output.Tokens.Refresh)
			assert.False(t, output.Tokens.ExpiresAt.IsZero())
		})
	}
}
