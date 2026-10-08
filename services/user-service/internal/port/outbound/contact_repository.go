package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type ContactRepository interface {
	Add(ctx context.Context, c contact.Contact) error
	GetByUserID(ctx context.Context, userID user.ID) ([]user.User, error)
}
