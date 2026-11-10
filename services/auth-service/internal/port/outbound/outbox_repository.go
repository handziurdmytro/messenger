package outbound

import (
	"context"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type OutboxRepository interface {
	SaveUserCreated(ctx context.Context, user auth.User) error
	SaveUserDeleted(ctx context.Context, userID auth.UserID, deletedAt time.Time) error
}
