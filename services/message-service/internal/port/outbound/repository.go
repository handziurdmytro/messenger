package outbound

import (
	"context"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type MessageRepository interface {
	Save(ctx context.Context, msg message.Message) error
	GetHistory(ctx context.Context, q message.Query) ([]message.Message, error)
}
