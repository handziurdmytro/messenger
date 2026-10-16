package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type ChatRepository interface {
	Save(ctx context.Context, c chat.Chat) error
	FindByID(ctx context.Context, id chat.ID) (chat.Chat, error)
	FindPrivateByParticipants(ctx context.Context, u1, u2 chat.UserID) (chat.Chat, error)
}
