package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type DeleteUserUseCase struct {
	users outbound.UserRepository
}

func NewDeleteUserUseCase(users outbound.UserRepository) *DeleteUserUseCase {
	return &DeleteUserUseCase{users: users}
}

func (uc *DeleteUserUseCase) Execute(ctx context.Context, id user.ID, deletedAt time.Time) error {
	u, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if err := u.Delete(deletedAt); err != nil {
		if errors.Is(err, user.ErrDeleted) {
			return nil
		}
		return fmt.Errorf("delete user: %w", err)
	}

	if err := uc.users.MarkDeleted(ctx, u); err != nil {
		return fmt.Errorf("mark user deleted: %w", err)
	}

	return nil
}
