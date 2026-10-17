package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/pkg/contracts/message"
)

type MessagePublisher interface {
	Publish(ctx context.Context, event message.CreatedEvent) error
}
