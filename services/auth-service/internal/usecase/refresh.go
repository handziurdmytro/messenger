package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

type RefreshUseCase struct {
	users    outbound.UserRepository
	tokens   outbound.TokenRepository
	tokenSvc *auth.TokenService
}

func NewRefreshUseCase(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	tokenSvc *auth.TokenService,
) *RefreshUseCase {
	return &RefreshUseCase{
		users:    users,
		tokens:   tokens,
		tokenSvc: tokenSvc,
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error) {
	entry, err := uc.tokens.FindRefresh(ctx, token)
	if err != nil {
		return auth.TokenPair{}, auth.ErrRefreshInvalid
	}

	if entry.IsExpired() {
		_ = uc.tokens.DeleteRefresh(ctx, token)
		return auth.TokenPair{}, auth.ErrRefreshExpired
	}

	user, err := uc.users.FindByID(ctx, entry.UserID)
	if err != nil {
		return auth.TokenPair{}, auth.ErrUserNotFound
	}

	if err := uc.tokens.DeleteRefresh(ctx, token); err != nil {
		return auth.TokenPair{}, fmt.Errorf("delete refresh: %w", err)
	}

	pair, newRefresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("issue tokens: %w", err)
	}

	if err := uc.tokens.SaveRefresh(ctx, newRefresh); err != nil {
		return auth.TokenPair{}, fmt.Errorf("save refresh: %w", err)
	}

	return pair, nil
}
