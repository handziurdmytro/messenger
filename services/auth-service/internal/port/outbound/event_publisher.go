package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type EventPublisher interface {
	PublishUserCreated(ctx context.Context, user auth.User) error
}
