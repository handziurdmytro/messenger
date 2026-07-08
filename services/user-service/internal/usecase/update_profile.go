package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type UpdateProfileUseCase struct {
	users outbound.UserRepository
}

func NewUpdateProfileUseCase(users outbound.UserRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, id user.ID, input user.UpdateInput) (user.User, error) {
	u, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("find user: %w", err)
	}

	if err := u.Apply(input); err != nil {
		return user.User{}, err
	}

	if err := uc.users.Update(ctx, u); err != nil {
		return user.User{}, fmt.Errorf("update user: %w", err)
	}

	return u, nil
}
