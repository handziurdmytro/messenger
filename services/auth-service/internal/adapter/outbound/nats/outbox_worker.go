package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handziurdmytro/messenger/pkg/natskit"
)

type OutboxWorker struct {
	db        *pgxpool.Pool
	publisher *natskit.Publisher
	log       *slog.Logger
	interval  time.Duration
}

func NewOutboxWorker(
	db *pgxpool.Pool,
	publisher *natskit.Publisher,
	log *slog.Logger,
	interval time.Duration,
) *OutboxWorker {
	return &OutboxWorker{db: db, publisher: publisher, log: log, interval: interval}
}

func (w *OutboxWorker) Run(ctx context.Context) error {
	if w.interval <= 0 {
		return fmt.Errorf("outbox polling interval must be positive")
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.publishPending(ctx); err != nil && ctx.Err() == nil {
			w.log.Error("outbox publish failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *OutboxWorker) publishPending(ctx context.Context) error {
	rows, err := w.db.Query(ctx, `
		SELECT id, subject, payload
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at, id
		LIMIT 100`)
	if err != nil {
		return fmt.Errorf("read outbox: %w", err)
	}

	type event struct {
		id      string
		subject string
		payload []byte
	}

	var events []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.id, &e.subject, &e.payload); err != nil {
			rows.Close()
			return fmt.Errorf("scan outbox: %w", err)
		}
		events = append(events, e)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("iterate outbox: %w", err)
	}

	for _, e := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.publisher.PublishWithID(ctx, e.subject, e.payload, e.id); err != nil {
			return fmt.Errorf("publish outbox %s: %w", e.id, err)
		}
		if _, err := w.db.Exec(ctx, `
			UPDATE outbox_events
			SET published_at = now()
			WHERE id = $1 AND published_at IS NULL`, e.id,
		); err != nil {
			return fmt.Errorf("mark outbox %s published: %w", e.id, err)
		}
	}
	return nil
}
