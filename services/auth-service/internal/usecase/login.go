package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type LoginUseCase struct {
	users    outbound.UserRepository
	tokens   outbound.TokenRepository
	tokenSvc TokenIssuer
}

func NewLoginUseCase(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	tokenSvc TokenIssuer,
) *LoginUseCase {
	return &LoginUseCase{
		users:    users,
		tokens:   tokens,
		tokenSvc: tokenSvc,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	Tokens auth.TokenPair
	UserID auth.UserID
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.users.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return LoginOutput{}, auth.ErrUserNotFound
		}

		return LoginOutput{}, errkit.Wrap(ErrInternal, "find user", err)
	}

	if err := auth.CheckPassword(input.Password, user.PasswordHash); err != nil {
		return LoginOutput{}, auth.ErrPasswordInvalid
	}

	pair, refresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return LoginOutput{}, errkit.Wrap(ErrInternal, "issue tokens", err)
	}

	if err := uc.tokens.SaveRefresh(ctx, refresh); err != nil {
		return LoginOutput{}, errkit.Wrap(ErrInternal, "save refresh", err)
	}

	return LoginOutput{
		Tokens: pair,
		UserID: user.ID,
	}, nil
}
