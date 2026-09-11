package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type mockUserRepository struct {
	findByEmailFn func(ctx context.Context, email string) (auth.User, error)
	findByIDFn    func(ctx context.Context, id auth.UserID) (auth.User, error)
	saveFn        func(ctx context.Context, user auth.User) error
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	return m.findByEmailFn(ctx, email)
}
func (m *mockUserRepository) Save(ctx context.Context, user auth.User) error {
	return m.saveFn(ctx, user)
}
func (m *mockUserRepository) FindByID(ctx context.Context, id auth.UserID) (auth.User, error) {
	return m.findByIDFn(ctx, id)
}

type mockTokenRepository struct {
	saveRefreshFn   func(ctx context.Context, entry auth.RefreshEntry) error
	findRefreshFn   func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error)
	deleteRefreshFn func(ctx context.Context, token auth.RefreshToken) error
}

func (m *mockTokenRepository) SaveRefresh(ctx context.Context, entry auth.RefreshEntry) error {
	return m.saveRefreshFn(ctx, entry)
}
func (m *mockTokenRepository) FindRefresh(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
	return m.findRefreshFn(ctx, token)
}
func (m *mockTokenRepository) DeleteRefresh(ctx context.Context, token auth.RefreshToken) error {
	return m.deleteRefreshFn(ctx, token)
}

type mockEventPublisher struct {
	publishUserCreatedFn func(ctx context.Context, user auth.User) error
}

func (m *mockEventPublisher) PublishUserCreated(ctx context.Context, user auth.User) error {
	return m.publishUserCreatedFn(ctx, user)
}
