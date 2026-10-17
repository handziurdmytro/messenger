package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type SendMessage struct {
	repo      MessageRepository
	publisher MessagePublisher
}

func NewSendMessage(repo MessageRepository, publisher MessagePublisher) *SendMessage {
	return &SendMessage{repo: repo, publisher: publisher}
}

func (uc *SendMessage) Execute(ctx context.Context, cmd contract.SendMessageCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	msg, err := message.New(
		message.ID(cmd.MessageID),
		message.ChatID(cmd.ChatID),
		cmd.FromUser,
		cmd.Payload,
		message.Type(cmd.Type),
	)
	if err != nil {
		return err
	}

	msg, err = uc.repo.Save(ctx, msg)
	if err != nil {
		return errkit.Wrap(ErrPersistence, "save message", err)
	}

	event := contract.CreatedEvent{
		MessageID: string(msg.ID),
		ChatID:    string(msg.ChatID),
		FromUser:  msg.FromUser,
		Type:      contract.Type(msg.Type),
		Payload:   msg.Payload,
		CreatedAt: msg.CreatedAt,
	}

	if err := uc.publisher.Publish(ctx, event); err != nil {
		return errkit.Wrap(ErrPublishFailed, "publish message created", err)
	}

	return nil
}
