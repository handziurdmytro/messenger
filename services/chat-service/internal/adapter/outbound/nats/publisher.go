package nats

import (
	"context"
	"encoding/json"

	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
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

func (p *MessagePublisher) Publish(ctx context.Context, cmd message.SendMessageCommand) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "marshal send message command", err)
	}

	if err := p.publisher.PublishWithID(ctx, message.SubjectSendMessage, data, cmd.MessageID); err != nil {
		return errkit.Wrap(usecase.ErrPublishFailed, "publish send message command", err)
	}

	return nil
}
