package usecase

import (
	"context"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type HistoryQuery struct {
	ChatID   message.ChatID
	Before   time.Time
	BeforeID message.ID
	Limit    int
}

type MessageRepository interface {
	Save(ctx context.Context, msg message.Message) (message.Message, error)
	FindByID(ctx context.Context, chatID message.ChatID, messageID message.ID) (message.Message, error)
	GetHistory(ctx context.Context, query HistoryQuery) ([]message.Message, error)
}
