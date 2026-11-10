package usecase

import (
	"context"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

const (
	DefaultChatsLimit = 50
	MaxChatsLimit     = 100
)

type ChatCursor struct {
	CreatedAt time.Time
	ID        chat.ID
}

type ChatPage struct {
	Limit  int
	Before *ChatCursor
}

type ChatLister interface {
	ListByUser(ctx context.Context, userID chat.UserID, page ChatPage) ([]chat.Chat, error)
}

type ListChatsUseCase struct{ repo ChatLister }

func NewListChats(repo ChatLister) *ListChatsUseCase { return &ListChatsUseCase{repo: repo} }

func (uc *ListChatsUseCase) Execute(ctx context.Context, id chat.UserID) ([]chat.Chat, error) {
	if _, err := chat.ParseUserID(string(id)); err != nil {
		return nil, err
	}
	return uc.repo.ListByUser(ctx, id, ChatPage{Limit: MaxChatsLimit})
}
