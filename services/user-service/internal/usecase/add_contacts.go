package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

var ErrCannotAddSelf = errors.New("cannot add yourself to contacts")

type AddContactUseCase struct {
	users outbound.UserRepository
}

func NewAddContactUseCase(users outbound.UserRepository) *AddContactUseCase {
	return &AddContactUseCase{users: users}
}

func (uc *AddContactUseCase) Execute(ctx context.Context, userID, contactID user.ID) error {
	if userID == contactID {
		return ErrCannotAddSelf
	}

	_, err := uc.users.FindByID(ctx, contactID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return user.ErrUserNotFound
		}
		return fmt.Errorf("check contact exists: %w", err)
	}

	if err := uc.users.AddContact(ctx, userID, contactID); err != nil {
		return fmt.Errorf("add contact: %w", err)
	}

	return nil
}
