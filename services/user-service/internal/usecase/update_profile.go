package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type UpdateProfileInput struct {
	DisplayName *string
	AvatarURL   *string
}

type UpdateProfileUseCase struct{ users outbound.UserRepository }

func NewUpdateProfileUseCase(users outbound.UserRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, id user.ID, input UpdateProfileInput) (user.User, error) {
	u, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("find user: %w", err)
	}

	if input.DisplayName == nil && input.AvatarURL == nil {
		return u, nil
	}

	now := time.Now().UTC()
	if input.DisplayName != nil {
		if err := u.ChangeDisplayName(*input.DisplayName, now); err != nil {
			return user.User{}, err
		}
	}

	if input.AvatarURL != nil {
		if err := u.ChangeAvatarURL(*input.AvatarURL, now); err != nil {
			return user.User{}, err
		}
	}

	if err := uc.users.Update(ctx, u); err != nil {
		return user.User{}, fmt.Errorf("update user: %w", err)
	}

	return u, nil
}
