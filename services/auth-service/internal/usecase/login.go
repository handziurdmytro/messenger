package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

type LoginUseCase struct {
	users    outbound.UserRepository
	tokens   outbound.TokenRepository
	tokenSvc *auth.TokenService
}

func NewLoginUseCase(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	tokenSvc *auth.TokenService,
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
		return LoginOutput{}, auth.ErrUserNotFound
	}

	if err := auth.CheckPassword(input.Password, user.PasswordHash); err != nil {
		return LoginOutput{}, auth.ErrPasswordInvalid
	}

	pair, refresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("%w: issue tokens: %v", ErrInternal, err)
	}

	if err := uc.tokens.SaveRefresh(ctx, refresh); err != nil {
		return LoginOutput{}, fmt.Errorf("%w: save refresh: %v", ErrInternal, err)
	}

	return LoginOutput{
		Tokens: pair,
		UserID: user.ID,
	}, nil
}
