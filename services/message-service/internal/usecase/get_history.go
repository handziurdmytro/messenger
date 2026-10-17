package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type GetHistory struct {
	repo MessageRepository
}

func NewGetHistory(repo MessageRepository) *GetHistory {
	return &GetHistory{repo: repo}
}

func (uc *GetHistory) Execute(ctx context.Context, q HistoryQuery) ([]message.Message, error) {
	if _, err := uuid.Parse(string(q.ChatID)); err != nil {
		return nil, ErrInvalidChatID
	}
	if q.BeforeID != "" {
		if q.Before.IsZero() {
			return nil, ErrInvalidHistoryCursor
		}
		if _, err := uuid.Parse(string(q.BeforeID)); err != nil {
			return nil, ErrInvalidHistoryCursor
		}
	}
	if q.Limit < 0 || q.Limit > 100 {
		return nil, ErrInvalidHistoryLimit
	}
	if q.Limit == 0 {
		q.Limit = 50
	}
	if q.Before.IsZero() {
		q.Before = time.Now().UTC()
	}

	return uc.repo.GetHistory(ctx, q)
}
