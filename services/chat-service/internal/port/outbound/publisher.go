package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type MessagePublisher interface {
	Publish(ctx context.Context, msg chat.Message) error
}
