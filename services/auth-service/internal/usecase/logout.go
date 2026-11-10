package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type LogoutUseCase struct {
	tokens outbound.TokenRepository
}

func NewLogoutUseCase(tokens outbound.TokenRepository) *LogoutUseCase {
	return &LogoutUseCase{tokens: tokens}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, token auth.RefreshToken) error {
	if token == "" {
		return auth.ErrRefreshInvalid
	}

	if err := uc.tokens.DeleteRefresh(ctx, token); err != nil {
		return errkit.Wrap(ErrInternal, "delete refresh", err)
	}

	return nil
}

type LogoutAllUseCase struct {
	tokens outbound.TokenRepository
}

func NewLogoutAllUseCase(tokens outbound.TokenRepository) *LogoutAllUseCase {
	return &LogoutAllUseCase{tokens: tokens}
}

func (uc *LogoutAllUseCase) Execute(ctx context.Context, userID auth.UserID) error {
	if userID == "" {
		return auth.ErrUserNotFound
	}

	if err := uc.tokens.DeleteAllRefresh(ctx, userID); err != nil {
		return errkit.Wrap(ErrInternal, "delete all refresh", err)
	}

	return nil
}
