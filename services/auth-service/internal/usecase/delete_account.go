package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type DeleteAccountUseCase struct {
	users      outbound.UserRepository
	transactor outbound.Transactor
}

func NewDeleteAccountUseCase(
	users outbound.UserRepository,
	transactor outbound.Transactor,
) *DeleteAccountUseCase {
	return &DeleteAccountUseCase{users: users, transactor: transactor}
}

type DeleteAccountInput struct {
	UserID   auth.UserID
	Password string
}

func (uc *DeleteAccountUseCase) Execute(ctx context.Context, input DeleteAccountInput) error {
	user, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return auth.ErrUserNotFound
		}

		return errkit.Wrap(ErrInternal, "find user", err)
	}

	if err := auth.CheckPassword(input.Password, user.PasswordHash); err != nil {
		return err
	}

	deletedAt := time.Now().UTC()
	err = uc.transactor.WithinTransaction(ctx, func(repos outbound.Repositories) error {
		if err := repos.Users.Delete(ctx, user.ID); err != nil {
			return err
		}

		return repos.Outbox.SaveUserDeleted(ctx, user.ID, deletedAt)
	})
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return auth.ErrUserNotFound
		}

		return errkit.Wrap(ErrInternal, "delete account", err)
	}

	return nil
}
