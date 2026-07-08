package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type GetProfileUseCase struct {
	users outbound.UserRepository
}

func NewGetProfileUseCase(users outbound.UserRepository) *GetProfileUseCase {
	return &GetProfileUseCase{users: users}
}

func (uc *GetProfileUseCase) Execute(ctx context.Context, id user.ID) (user.User, error) {
	u, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("get profile: %w", err)
	}
	return u, nil
}
