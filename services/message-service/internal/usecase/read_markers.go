package usecase

import (
	"context"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type ReadMarker struct {
	UserID    string
	MessageID message.ID
	ReadAt    time.Time
}

type ReadMarkerRepository interface {
	MarkRead(ctx context.Context, chatID message.ChatID, marker ReadMarker) (ReadMarker, error)
	ListReadMarkers(ctx context.Context, chatID message.ChatID) ([]ReadMarker, error)
	CountUnread(ctx context.Context, chatID message.ChatID, userID string, since time.Time, limit int) (int, error)
}
