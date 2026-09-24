package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type TokenIssuer interface {
	Issue(auth.User) (auth.TokenPair, auth.RefreshEntry, error)
}

type RegisterUseCase struct {
	transactor outbound.Transactor
	tokenSvc   TokenIssuer
	publisher  outbound.EventPublisher
}

func NewRegisterUseCase(
	transactor outbound.Transactor,
	tokenSvc TokenIssuer,
	publisher outbound.EventPublisher,
) *RegisterUseCase {
	return &RegisterUseCase{
		transactor: transactor,
		tokenSvc:   tokenSvc,
		publisher:  publisher,
	}
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

		return repos.Tokens.SaveRefresh(ctx, refresh)
	})
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			return RegisterOutput{}, auth.ErrEmailTaken
		}

		return RegisterOutput{}, errkit.Wrap(ErrInternal, "persist registration", err)
	}

	if err := uc.publisher.PublishUserCreated(ctx, user); err != nil {
		logger.WithComponent(logger.FromContext(ctx), "auth").Error(
			"publish user.created failed",
			logger.KeyUserID, user.ID,
			logger.KeyError, err,
		)
	}

	return RegisterOutput{
		Tokens: pair,
		UserID: user.ID,
	}, nil
}
