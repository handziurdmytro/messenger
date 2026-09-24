package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type RefreshUseCase struct {
	users      outbound.UserRepository
	tokens     outbound.TokenRepository
	transactor outbound.Transactor
	tokenSvc   TokenIssuer
}

func NewRefreshUseCase(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	transactor outbound.Transactor,
	tokenSvc TokenIssuer,
) *RefreshUseCase {
	return &RefreshUseCase{
		users:      users,
		tokens:     tokens,
		transactor: transactor,
		tokenSvc:   tokenSvc,
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, token auth.RefreshToken) (auth.TokenPair, error) {
	entry, err := uc.tokens.FindRefresh(ctx, token)
	if err != nil {
		if errors.Is(err, auth.ErrRefreshInvalid) {
			return auth.TokenPair{}, auth.ErrRefreshInvalid
		}

		return auth.TokenPair{}, errkit.Wrap(ErrInternal, "find refresh", err)
	}

	if entry.IsExpired() {
		if err := uc.tokens.DeleteRefresh(ctx, token); err != nil {
			return auth.TokenPair{}, errkit.Wrap(ErrInternal, "delete expired refresh", err)
		}

		return auth.TokenPair{}, auth.ErrRefreshExpired
	}

	user, err := uc.users.FindByID(ctx, entry.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return auth.TokenPair{}, auth.ErrUserNotFound
		}

		return auth.TokenPair{}, errkit.Wrap(ErrInternal, "find user", err)
	}

	pair, newRefresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return auth.TokenPair{}, errkit.Wrap(ErrInternal, "issue tokens", err)
	}

	err = uc.transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		if err := repos.Tokens.DeleteRefresh(ctx, token); err != nil {
			return err
		}

		return repos.Tokens.SaveRefresh(ctx, newRefresh)
	})
	if err != nil {
		return auth.TokenPair{}, errkit.Wrap(ErrInternal, "rotate refresh token", err)
	}

	return pair, nil
}
