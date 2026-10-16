package usecase

import (
	"context"
	"encoding/json"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
)

type SendMessageInput struct {
	MessageID string
	ChatID    chat.ID
	FromUser  chat.UserID
	Type      message.Type
	Payload   json.RawMessage
}

type MessagePublisher interface {
	Publish(ctx context.Context, cmd message.SendMessageCommand) error
}

type SendMessageUseCase struct {
	repo      ChatRepository
	publisher MessagePublisher
}

func NewSendMessage(repo ChatRepository, publisher MessagePublisher) *SendMessageUseCase {
	return &SendMessageUseCase{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *SendMessageUseCase) Execute(ctx context.Context, in SendMessageInput) (string, error) {
	chatID, err := chat.ParseID(string(in.ChatID))
	if err != nil {
		return "", err
	}

	fromUser, err := chat.ParseUserID(string(in.FromUser))
	if err != nil {
		return "", err
	}

	cmd := message.SendMessageCommand{
		MessageID: in.MessageID,
		ChatID:    string(chatID),
		FromUser:  string(fromUser),
		Type:      in.Type,
		Payload:   in.Payload,
	}

	if err := cmd.Validate(); err != nil {
		return "", err
	}

	c, err := uc.repo.FindByID(ctx, chatID)
	if err != nil {
		return "", err
	}

	if !c.HasParticipant(fromUser) {
		return "", ErrForbidden
	}

	if err := uc.publisher.Publish(ctx, cmd); err != nil {
		return "", err
	}

	return cmd.MessageID, nil
}
