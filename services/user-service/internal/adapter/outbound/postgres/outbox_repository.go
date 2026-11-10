package postgres

import (
	"context"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/natskit"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type OutboxRepository struct {
	q *db.Queries
}

func NewOutboxRepository(q *db.Queries) *OutboxRepository {
	return &OutboxRepository{q: q}
}

func (r *OutboxRepository) ListPending(ctx context.Context, limit int32) ([]natskit.OutboxMessage, error) {
	rows, err := r.q.ListPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "list pending outbox events", err)
	}

	messages := make([]natskit.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, natskit.OutboxMessage{
			ID:      row.ID,
			Subject: row.Subject,
			Payload: row.Payload,
		})
	}
	return messages, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	if err := r.q.MarkOutboxEventPublished(ctx, id); err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "mark outbox event published", err)
	}
	return nil
}
