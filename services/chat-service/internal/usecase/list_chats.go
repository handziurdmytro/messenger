package usecase

import (
	"context"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type ChatLister interface {
	ListByUser(context.Context, chat.UserID) ([]chat.Chat, error)
}

type ListChatsUseCase struct{ repo ChatLister }

func NewListChats(repo ChatLister) *ListChatsUseCase { return &ListChatsUseCase{repo: repo} }

func (uc *ListChatsUseCase) Execute(ctx context.Context, id chat.UserID) ([]chat.Chat, error) {
	if _, err := chat.ParseUserID(string(id)); err != nil {
		return nil, err
	}
	return uc.repo.ListByUser(ctx, id)
}
