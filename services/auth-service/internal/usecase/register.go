package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type TokenIssuer interface {
	Issue(auth.User) (auth.TokenPair, auth.RefreshEntry, error)
}

type RegisterUseCase struct {
	transactor outbound.Transactor
	tokenSvc   TokenIssuer
}

func NewRegisterUseCase(
	transactor outbound.Transactor,
	tokenSvc TokenIssuer,
) *RegisterUseCase {
	return &RegisterUseCase{transactor: transactor, tokenSvc: tokenSvc}
}

type RegisterInput struct {
	Email    string
	Password string
}

type RegisterOutput struct {
	Tokens auth.TokenPair
	UserID auth.UserID
}

func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	user, err := auth.NewUser(input.Email, input.Password)
	if err != nil {
		return RegisterOutput{}, errkit.Wrap(ErrInternal, "create user", err)
	}

	pair, refresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return RegisterOutput{}, errkit.Wrap(ErrInternal, "issue tokens", err)
	}

	err = uc.transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		if err := repos.Users.Save(ctx, user); err != nil {
			return err
		}
		if err := repos.Tokens.SaveRefresh(ctx, refresh); err != nil {
			return err
		}
		return repos.Outbox.SaveUserCreated(ctx, user)
	})
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			return RegisterOutput{}, auth.ErrEmailTaken
		}
		return RegisterOutput{}, errkit.Wrap(ErrInternal, "persist registration", err)
	}

	return RegisterOutput{Tokens: pair, UserID: user.ID}, nil
}
