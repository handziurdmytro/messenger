package postgres

import (
	"context"
	"encoding/json"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type OutboxRepository struct {
	q *db.Queries
}

func NewOutboxRepository(q *db.Queries) *OutboxRepository {
	return &OutboxRepository{q: q}
}

func (r *OutboxRepository) SaveUserCreated(ctx context.Context, user auth.User) error {
	event := contract.CreatedEvent{
		UserID:    string(user.ID),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
	if err := event.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "marshal user created", err)
	}

	if err := r.q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		ID:      event.UserID,
		Subject: contract.SubjectCreated,
		Payload: data,
	}); err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "save outbox event", err)
	}
	return nil
}

func (r *OutboxRepository) ListPending(ctx context.Context, limit int32) ([]outbound.OutboxEvent, error) {
	rows, err := r.q.ListPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "list pending outbox events", err)
	}

	events := make([]outbound.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, outbound.OutboxEvent{
			ID:      row.ID,
			Subject: row.Subject,
			Payload: row.Payload,
		})
	}
	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	if err := r.q.MarkOutboxEventPublished(ctx, id); err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "mark outbox event published", err)
	}
	return nil
}
