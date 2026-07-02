package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type UserRepository interface {
	Save(ctx context.Context, user auth.User) error
	FindByEmail(ctx context.Context, email string) (auth.User, error)
	FindByID(ctx context.Context, id auth.UserID) (auth.User, error)
}
