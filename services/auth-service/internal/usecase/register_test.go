package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

func TestRegisterUseCase_Execute(t *testing.T) {
	testPEM := generateTestRSAKeyPEM(t)
	tokenSvc, _ := auth.NewTokenService(testPEM, 15*time.Minute, 24*time.Hour)

	tests := []struct {
		name       string
		input      RegisterInput
		setupMocks func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher)
		wantErr    error
	}{
		{
			name:  "success",
			input: RegisterInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher) {
				users.saveFn = func(ctx context.Context, user auth.User) error {
					return nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return nil
				}
				publisher.publishUserCreatedFn = func(ctx context.Context, user auth.User) error {
					return nil
				}
			},
			wantErr: nil,
		},
		{
			name:       "invalid input (short password)",
			input:      RegisterInput{Email: "test@gmail.com", Password: "123"},
			setupMocks: nil,
			wantErr:    auth.ErrPasswordTooShort,
		},
		{
			name:  "email already taken",
			input: RegisterInput{Email: "taken@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher) {
				users.saveFn = func(ctx context.Context, user auth.User) error {
					return auth.ErrEmailTaken
				}
			},
			wantErr: auth.ErrEmailTaken,
		},
		{
			name:  "database fail on save user",
			input: RegisterInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher) {
				users.saveFn = func(ctx context.Context, user auth.User) error {
					return errors.New("db connection lost")
				}
			},
			wantErr: ErrInternal,
		},
		{
			name:  "database fail on save refresh",
			input: RegisterInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher) {
				users.saveFn = func(ctx context.Context, user auth.User) error {
					return nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return errors.New("redis timeout")
				}
			},
			wantErr: ErrInternal,
		},
		{
			name:  "publisher fails but usecase succeeds",
			input: RegisterInput{Email: "test@gmail.com", Password: "pass1234"},
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository, publisher *mockEventPublisher) {
				users.saveFn = func(ctx context.Context, user auth.User) error {
					return nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return nil
				}
				publisher.publishUserCreatedFn = func(ctx context.Context, user auth.User) error {
					return errors.New("kafka broker down")
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usersMock := &mockUserRepository{}
			tokensMock := &mockTokenRepository{}
			publisherMock := &mockEventPublisher{}

			if tt.setupMocks != nil {
				tt.setupMocks(usersMock, tokensMock, publisherMock)
			}

			uc := NewRegisterUseCase(usersMock, tokensMock, tokenSvc, publisherMock)

			_, err := uc.Execute(context.Background(), tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
