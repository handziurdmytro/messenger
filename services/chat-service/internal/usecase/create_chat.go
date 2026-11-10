package usecase

import (
	"context"
	"errors"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type BlockChecker interface {
	ExistsBetween(ctx context.Context, first, second chat.UserID) (bool, error)
}

type CreateChatUseCase struct {
	repo   ChatRepository
	blocks BlockChecker
}

func NewCreateChat(repo ChatRepository, blocks BlockChecker) *CreateChatUseCase {
	return &CreateChatUseCase{repo: repo, blocks: blocks}
}

func (uc *CreateChatUseCase) Execute(ctx context.Context, u1, u2 chat.UserID) (chat.Chat, error) {
	c, err := chat.NewPrivate(u1, u2)
	if err != nil {
		return chat.Chat{}, err
	}

	if err := ensureNotBlocked(ctx, uc.blocks, u1, u2); err != nil {
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

func ensureNotBlocked(ctx context.Context, blocks BlockChecker, first, second chat.UserID) error {
	blocked, err := blocks.ExistsBetween(ctx, first, second)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocked
	}
	return nil
}
