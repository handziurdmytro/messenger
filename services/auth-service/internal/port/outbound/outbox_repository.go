package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type OutboxRepository interface {
	SaveUserCreated(ctx context.Context, user auth.User) error
}
