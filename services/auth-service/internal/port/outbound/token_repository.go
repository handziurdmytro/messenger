package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
)

type TokenRepository interface {
	SaveRefresh(ctx context.Context, entry auth.RefreshEntry) error
	FindRefresh(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error)
	DeleteRefresh(ctx context.Context, token auth.RefreshToken) error
}
