package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type MessagePublisher interface {
	Publish(ctx context.Context, msg message.Message) error
}
