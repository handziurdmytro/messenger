package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type UserRepository interface {
	Save(ctx context.Context, u user.User) error
	FindByID(ctx context.Context, id user.ID) (user.User, error)
	FindByIDs(ctx context.Context, ids []user.ID) ([]user.User, error)
	Update(ctx context.Context, u user.User) error
	Search(ctx context.Context, query string, limit int) ([]user.User, error)
	AddContact(ctx context.Context, userID, contactID user.ID) error
	GetContacts(ctx context.Context, userID user.ID) ([]user.User, error)
}
