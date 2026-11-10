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

	switch c.Type() {
	case chat.TypePrivate:
		return uc.repo.Delete(ctx, chatID)
	case chat.TypeGroup:
		if err := c.Leave(userID); err != nil {
			return err
		}
		if c.IsEmpty() {
			return uc.repo.Delete(ctx, chatID)
		}
		return uc.repo.Update(ctx, c)
	default:
		return chat.ErrUnsupportedChatType
	}
}
