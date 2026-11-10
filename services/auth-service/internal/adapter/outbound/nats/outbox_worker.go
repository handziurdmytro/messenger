package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

const outboxBatchSize int32 = 100

type Publisher interface {
	PublishWithID(ctx context.Context, subject string, data []byte, id string) error
}

type OutboxWorker struct {
	store     outbound.OutboxStore
	publisher Publisher
	log       *slog.Logger
	interval  time.Duration
}

func NewOutboxWorker(
	store outbound.OutboxStore,
	publisher Publisher,
	log *slog.Logger,
	interval time.Duration,
) *OutboxWorker {
	return &OutboxWorker{store: store, publisher: publisher, log: log, interval: interval}
}

func (w *OutboxWorker) Run(ctx context.Context) error {
	if w.interval <= 0 {
		return errors.New("outbox polling interval must be positive")
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
	events, err := w.store.ListPending(ctx, outboxBatchSize)
	if err != nil {
		return fmt.Errorf("read outbox: %w", err)
	}

	for _, e := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.publisher.PublishWithID(ctx, e.Subject, e.Payload, e.ID); err != nil {
			return fmt.Errorf("publish outbox %s: %w", e.ID, err)
		}
		if err := w.store.MarkPublished(ctx, e.ID); err != nil {
			return fmt.Errorf("mark outbox %s published: %w", e.ID, err)
		}
	}
	return nil
}
