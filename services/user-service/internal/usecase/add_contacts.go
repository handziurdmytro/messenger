package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type AddContactUseCase struct {
	users    outbound.UserRepository
	contacts outbound.ContactRepository
}

func NewAddContactUseCase(users outbound.UserRepository, contacts outbound.ContactRepository) *AddContactUseCase {
	return &AddContactUseCase{users: users, contacts: contacts}
}

func (uc *AddContactUseCase) Execute(ctx context.Context, userID, contactID user.ID) error {
	c, err := contact.New(userID, contactID, time.Now().UTC())
	if err != nil {
		return err
	}
	if _, err := uc.users.FindByID(ctx, contactID); err != nil {
		return fmt.Errorf("check contact exists: %w", err)
	}
	if err := uc.contacts.Add(ctx, c); err != nil {
		return fmt.Errorf("add contact: %w", err)
	}
	return nil
}
