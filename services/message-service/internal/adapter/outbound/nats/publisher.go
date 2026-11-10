package nats

import (
	"context"
	"encoding/json"
	"fmt"

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
	if err := p.publisher.PublishWithID(ctx, message.SubjectMessageCreated, data, event.MessageID+":created"); err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "publish message created event", err)
	}
	return nil
}

func (p *MessagePublisher) PublishEdited(ctx context.Context, event message.EditedEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	id := fmt.Sprintf("%s:edited:%d", event.MessageID, event.EditedAt.UnixNano())
	return p.publish(ctx, message.SubjectMessageEdited, id, event)
}

func (p *MessagePublisher) PublishDeleted(ctx context.Context, event message.DeletedEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	return p.publish(ctx, message.SubjectMessageDeleted, event.MessageID+":deleted", event)
}

func (p *MessagePublisher) publish(ctx context.Context, subject, id string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "marshal "+subject, err)
	}
	if err := p.publisher.PublishWithID(ctx, subject, data, id); err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "publish "+subject, err)
	}
	return nil
}
