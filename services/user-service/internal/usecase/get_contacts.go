package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const (
	defaultContactsLimit = 100
	maxContactsLimit     = 200
)

type GetContactsUseCase struct {
	contacts outbound.ContactRepository
}

func NewGetContactsUseCase(contacts outbound.ContactRepository) *GetContactsUseCase {
	return &GetContactsUseCase{contacts: contacts}
}

func (uc *GetContactsUseCase) Execute(ctx context.Context, userID user.ID, limit, offset int) ([]user.User, error) {
	if limit <= 0 || limit > maxContactsLimit {
		limit = defaultContactsLimit
	}

	contacts, err := uc.contacts.GetByUserID(ctx, userID, outbound.Page{Limit: limit, Offset: max(offset, 0)})
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}

	return contacts, nil
}
