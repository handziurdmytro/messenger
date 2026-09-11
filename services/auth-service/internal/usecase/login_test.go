package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

func TestLoginUseCase_Execute(t *testing.T) {
	testPEM := generateTestRSAKeyPEM(t)
	tokenSvc, _ := auth.NewTokenService(testPEM, 15*time.Minute, 24*time.Hour)

	validPassword := "pass1234"
	testUser, _ := auth.NewUser("test@gmail.com", validPassword)

	tests := []struct {
		name       string
		input      LoginInput
		setupMocks func(users *mockUserRepository, tokens *mockTokenRepository)
		wantErr    error
	}{
		{
			name:  "user not found",
			input: LoginInput{Email: "wrong@gmail.com", Password: "123"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				users.findByEmailFn = func(ctx context.Context, email string) (auth.User, error) {
					return auth.User{}, auth.ErrUserNotFound
				}
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:  "invalid password",
			input: LoginInput{Email: "test@gmail.com", Password: "pass4321"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				users.findByEmailFn = func(ctx context.Context, email string) (auth.User, error) {
					return testUser, nil
				}
			},
			wantErr: auth.ErrPasswordInvalid,
		},
		{
			name:  "save refresh failed",
			input: LoginInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				users.findByEmailFn = func(ctx context.Context, email string) (auth.User, error) {
					return testUser, nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return errors.New("database connection lost")
				}
			},
			wantErr: ErrInternal,
		},
		{
			name:  "success",
			input: LoginInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				users.findByEmailFn = func(ctx context.Context, email string) (auth.User, error) {
					return testUser, nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return nil
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usersMock := &mockUserRepository{}
			tokensMock := &mockTokenRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(usersMock, tokensMock)
			}

			uc := NewLoginUseCase(usersMock, tokensMock, tokenSvc)

			_, err := uc.Execute(context.Background(), tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
