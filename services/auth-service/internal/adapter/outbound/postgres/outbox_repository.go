package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type OutboxRepository struct {
	tx pgx.Tx
}

func NewOutboxRepository(tx pgx.Tx) *OutboxRepository {
	return &OutboxRepository{tx: tx}
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

	_, err = r.tx.Exec(ctx, `
		INSERT INTO outbox_events (id, subject, payload)
		VALUES ($1, $2, $3)`,
		event.UserID, contract.SubjectCreated, data,
	)
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "save outbox event", err)
	}
	return nil
}
