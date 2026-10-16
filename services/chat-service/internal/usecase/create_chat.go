package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type CreateChatUseCase struct {
	repo ChatRepository
}

func NewCreateChat(repo ChatRepository) *CreateChatUseCase {
	return &CreateChatUseCase{repo: repo}
}

func (uc *CreateChatUseCase) Execute(ctx context.Context, u1, u2 chat.UserID) (chat.Chat, error) {
	c, err := chat.NewPrivate(u1, u2)
	if err != nil {
		return chat.Chat{}, err
	}

	if err := uc.repo.Save(ctx, c); err != nil {
		if errors.Is(err, ErrChatAlreadyExists) {
			return uc.repo.FindPrivateByParticipants(ctx, u1, u2)
		}
		return chat.Chat{}, err
	}

	return c, nil
}
