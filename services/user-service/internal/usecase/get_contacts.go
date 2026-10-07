package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type GetContactsUseCase struct {
	users outbound.UserRepository
}

func NewGetContactsUseCase(users outbound.UserRepository) *GetContactsUseCase {
	return &GetContactsUseCase{users: users}
}

func (uc *GetContactsUseCase) Execute(ctx context.Context, userID user.ID) ([]user.User, error) {
	contacts, err := uc.users.GetContacts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}

	return contacts, nil
}
