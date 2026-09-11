package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

func TestRefreshUseCase_Execute(t *testing.T) {

	testPEM := generateTestRSAKeyPEM(t)
	tokenSvc, _ := auth.NewTokenService(testPEM, 15*time.Minute, 24*time.Hour)

	tests := []struct {
		name       string
		token      auth.RefreshToken
		setupMocks func(users *mockUserRepository, tokens *mockTokenRepository)
		wantErr    error
	}{
		{
			name:  "success",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				users.findByIDFn = func(ctx context.Context, id auth.UserID) (auth.User, error) {
					return auth.User{}, nil
				}

				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{
						ExpiresAt: time.Now().Add(time.Minute),
						CreatedAt: time.Now(),
					}, nil
				}

				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return nil
				}

				tokens.deleteRefreshFn = func(ctx context.Context, token auth.RefreshToken) error {
					return nil
				}
			},
			wantErr: nil,
		},
		{
			name:  "invalid refresh",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{}, errors.New("database connection lost")
				}
			},
			wantErr: auth.ErrRefreshInvalid,
		},
		{
			name:  "expired",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{
						CreatedAt: time.Now(),
						ExpiresAt: time.Now().Add(-time.Minute),
					}, nil
				}
				tokens.deleteRefreshFn = func(ctx context.Context, token auth.RefreshToken) error {
					return nil
				}
			},
			wantErr: auth.ErrRefreshExpired,
		},
		{
			name:  "fail delete refresh",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{
						ExpiresAt: time.Now().Add(time.Minute),
						CreatedAt: time.Now(),
					}, nil
				}
				users.findByIDFn = func(ctx context.Context, id auth.UserID) (auth.User, error) {
					return auth.User{}, nil
				}
				tokens.deleteRefreshFn = func(ctx context.Context, token auth.RefreshToken) error {
					return errors.New("database connection lost")
				}
			},
			wantErr: ErrInternal,
		},
		{
			name:  "user not found",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{
						ExpiresAt: time.Now().Add(time.Minute),
						CreatedAt: time.Now(),
					}, nil
				}
				tokens.deleteRefreshFn = func(ctx context.Context, token auth.RefreshToken) error {
					return nil
				}
				users.findByIDFn = func(ctx context.Context, id auth.UserID) (auth.User, error) {
					return auth.User{}, auth.ErrUserNotFound
				}
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:  "fail save refresh",
			token: "valid-token",
			setupMocks: func(users *mockUserRepository, tokens *mockTokenRepository) {
				tokens.findRefreshFn = func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
					return auth.RefreshEntry{
						ExpiresAt: time.Now().Add(time.Minute),
						CreatedAt: time.Now(),
					}, nil
				}
				users.findByIDFn = func(ctx context.Context, id auth.UserID) (auth.User, error) {
					return auth.User{}, nil
				}
				tokens.deleteRefreshFn = func(ctx context.Context, token auth.RefreshToken) error {
					return nil
				}
				tokens.saveRefreshFn = func(ctx context.Context, entry auth.RefreshEntry) error {
					return errors.New("database connection lost")
				}
			},
			wantErr: ErrInternal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usersMock := &mockUserRepository{}
			tokensMock := &mockTokenRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(usersMock, tokensMock)
			}
			uc := NewRefreshUseCase(usersMock, tokensMock, tokenSvc)

			_, err := uc.Execute(context.Background(), tt.token)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
