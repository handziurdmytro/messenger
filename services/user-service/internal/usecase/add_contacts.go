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
	blocks   outbound.BlockRepository
}

func NewAddContactUseCase(
	users outbound.UserRepository,
	contacts outbound.ContactRepository,
	blocks outbound.BlockRepository,
) *AddContactUseCase {
	return &AddContactUseCase{users: users, contacts: contacts, blocks: blocks}
}

func (uc *AddContactUseCase) Execute(ctx context.Context, userID, contactID user.ID) error {
	c, err := contact.New(userID, contactID, time.Now().UTC())
	if err != nil {
		return err
	}
	target, err := uc.users.FindByID(ctx, contactID)
	if err != nil {
		return fmt.Errorf("check contact exists: %w", err)
	}
	if target.IsDeleted() {
		return fmt.Errorf("check contact exists: %w", user.ErrNotFound)
	}

	blocked, err := uc.blocks.ExistsBetween(ctx, userID, contactID)
	if err != nil {
		return fmt.Errorf("check block: %w", err)
	}
	if blocked {
		return contact.ErrBlocked
	}
	if err := uc.contacts.Add(ctx, c); err != nil {
		return fmt.Errorf("add contact: %w", err)
	}
	return nil
}
