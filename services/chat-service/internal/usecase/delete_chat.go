package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type DeleteChatUseCase struct {
	repo ChatRepository
}

func NewDeleteChat(repo ChatRepository) *DeleteChatUseCase {
	return &DeleteChatUseCase{repo: repo}
}

func (uc *DeleteChatUseCase) Execute(ctx context.Context, chatID chat.ID, userID chat.UserID) error {
	if _, err := chat.ParseID(string(chatID)); err != nil {
		return err
	}
	if _, err := chat.ParseUserID(string(userID)); err != nil {
		return err
	}

	c, err := uc.repo.FindByID(ctx, chatID)
	if err != nil {
		return err
	}

	if !c.HasParticipant(userID) {
		return ErrForbidden
	}

	if c.Type() != chat.TypePrivate {
		return chat.ErrUnsupportedChatType
	}

	return uc.repo.Delete(ctx, chatID)
}
