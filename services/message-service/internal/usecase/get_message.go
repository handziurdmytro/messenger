package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type GetMessage struct {
	repo MessageRepository
}

func NewGetMessage(repo MessageRepository) *GetMessage {
	return &GetMessage{repo: repo}
}

func (uc *GetMessage) Execute(ctx context.Context, chatID message.ChatID, messageID message.ID) (message.Message, error) {
	if _, err := uuid.Parse(string(chatID)); err != nil {
		return message.Message{}, ErrInvalidChatID
	}
	if _, err := uuid.Parse(string(messageID)); err != nil {
		return message.Message{}, ErrInvalidMessageID
	}
	return uc.repo.FindByID(ctx, chatID, messageID)
}
