package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type RemoveContactUseCase struct {
	contacts outbound.ContactRepository
}

func NewRemoveContactUseCase(contacts outbound.ContactRepository) *RemoveContactUseCase {
	return &RemoveContactUseCase{contacts: contacts}
}

func (uc *RemoveContactUseCase) Execute(ctx context.Context, userID, contactID user.ID) error {
	if userID.IsZero() {
		return contact.ErrUserIDEmpty
	}
	if contactID.IsZero() {
		return contact.ErrContactIDEmpty
	}

	if err := uc.contacts.Remove(ctx, userID, contactID); err != nil {
		return fmt.Errorf("remove contact: %w", err)
	}

	return nil
}
