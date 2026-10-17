package nats

import (
	"context"
	"encoding/json"

	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/natskit"
)

type MessagePublisher struct {
	publisher *natskit.Publisher
}

func NewMessagePublisher(publisher *natskit.Publisher) *MessagePublisher {
	return &MessagePublisher{publisher: publisher}
}

func (p *MessagePublisher) Publish(ctx context.Context, event message.CreatedEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "marshal message created event", err)
	}
	if err := p.publisher.PublishWithID(ctx, message.SubjectMessageCreated, data, event.MessageID); err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "publish message created event", err)
	}
	return nil
}
