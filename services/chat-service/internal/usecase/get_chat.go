package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type GetChatUseCase struct {
	repo ChatRepository
}

func NewGetChat(repo ChatRepository) *GetChatUseCase {
	return &GetChatUseCase{repo: repo}
}

func (uc *GetChatUseCase) Execute(ctx context.Context, chatID chat.ID, userID chat.UserID) (chat.Chat, error) {
	if _, err := chat.ParseID(string(chatID)); err != nil {
		return chat.Chat{}, err
	}
	if _, err := chat.ParseUserID(string(userID)); err != nil {
		return chat.Chat{}, err
	}

	c, err := uc.repo.FindByID(ctx, chatID)
	if err != nil {
		return chat.Chat{}, err
	}

	if !c.HasParticipant(userID) {
		return chat.Chat{}, ErrForbidden
	}

	return c, nil
}
