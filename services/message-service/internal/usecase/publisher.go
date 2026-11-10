package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/pkg/contracts/message"
)

type MessagePublisher interface {
	Publish(ctx context.Context, event message.CreatedEvent) error
	PublishEdited(ctx context.Context, event message.EditedEvent) error
	PublishDeleted(ctx context.Context, event message.DeletedEvent) error
}
