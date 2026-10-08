package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type GetContactsUseCase struct {
	contacts outbound.ContactRepository
}

func NewGetContactsUseCase(contacts outbound.ContactRepository) *GetContactsUseCase {
	return &GetContactsUseCase{contacts: contacts}
}

func (uc *GetContactsUseCase) Execute(ctx context.Context, userID user.ID) ([]user.User, error) {
	contacts, err := uc.contacts.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}

	return contacts, nil
}
