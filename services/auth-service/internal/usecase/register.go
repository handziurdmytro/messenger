package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

type RegisterUseCase struct {
	users     outbound.UserRepository
	tokens    outbound.TokenRepository
	tokenSvc  *auth.TokenService
	publisher outbound.EventPublisher
}

func NewRegisterUseCase(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	tokenSvc *auth.TokenService,
	publisher outbound.EventPublisher,
) *RegisterUseCase {
	return &RegisterUseCase{
		users:     users,
		tokens:    tokens,
		tokenSvc:  tokenSvc,
		publisher: publisher,
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
		return RegisterOutput{}, fmt.Errorf("create user: %w", err)
	}

	if err := uc.users.Save(ctx, user); err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			return RegisterOutput{}, auth.ErrEmailTaken
		}
		return RegisterOutput{}, fmt.Errorf("save user: %w", err)
	}

	pair, refresh, err := uc.tokenSvc.Issue(user)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("issue tokens: %w", err)
	}

	if err := uc.tokens.SaveRefresh(ctx, refresh); err != nil {
		return RegisterOutput{}, fmt.Errorf("save refresh: %w", err)
	}

	if err := uc.publisher.PublishUserCreated(ctx, user); err != nil {
		slog.Error("publish user.created failed", "error", err)
	}

	return RegisterOutput{
		Tokens: pair,
		UserID: user.ID,
	}, nil
}
