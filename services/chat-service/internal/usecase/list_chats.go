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

type ChatList struct {
	Chats []chat.Chat
	Next  *ChatCursor
}

func (uc *ListChatsUseCase) Execute(ctx context.Context, id chat.UserID, limit int, before *ChatCursor) (ChatList, error) {
	if _, err := chat.ParseUserID(string(id)); err != nil {
		return ChatList{}, err
	}

	if limit <= 0 || limit > MaxChatsLimit {
		limit = DefaultChatsLimit
	}

	chats, err := uc.repo.ListByUser(ctx, id, ChatPage{Limit: limit + 1, Before: before})
	if err != nil {
		return ChatList{}, err
	}

	if len(chats) <= limit {
		return ChatList{Chats: chats}, nil
	}

	chats = chats[:limit]
	last := chats[len(chats)-1]
	return ChatList{
		Chats: chats,
		Next:  &ChatCursor{CreatedAt: last.CreatedAt(), ID: last.ID()},
	}, nil
}
