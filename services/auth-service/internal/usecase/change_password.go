package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type ChangePasswordUseCase struct {
	users      outbound.UserRepository
	transactor outbound.Transactor
}

func NewChangePasswordUseCase(
	users outbound.UserRepository,
	transactor outbound.Transactor,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{users: users, transactor: transactor}
}

type ChangePasswordInput struct {
	UserID          auth.UserID
	CurrentPassword string
	NewPassword     string
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) error {
	user, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return auth.ErrUserNotFound
		}

		return errkit.Wrap(ErrInternal, "find user", err)
	}

	hash, err := auth.ChangePassword(user.PasswordHash, input.CurrentPassword, input.NewPassword)
	if err != nil {
		return err
	}

	err = uc.transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		if err := repos.Users.UpdatePassword(ctx, user.ID, hash); err != nil {
			return err
		}

		return repos.Tokens.DeleteAllRefresh(ctx, user.ID)
	})
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return auth.ErrUserNotFound
		}

		return errkit.Wrap(ErrInternal, "update password", err)
	}

	return nil
}
