package natskit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const DefaultOutboxBatchSize int32 = 100

var ErrInvalidOutboxConfig = errors.New("invalid outbox relay config")

type OutboxMessage struct {
	ID      string
	Subject string
	Payload []byte
}

type OutboxStore interface {
	ListPending(ctx context.Context, limit int32) ([]OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
}

type MessagePublisher interface {
	PublishWithID(ctx context.Context, subject string, data []byte, id string) error
}

type OutboxRelay struct {
	store     OutboxStore
	publisher MessagePublisher
	log       *slog.Logger
	interval  time.Duration
	batchSize int32
}

func NewOutboxRelay(
	store OutboxStore,
	publisher MessagePublisher,
	log *slog.Logger,
	interval time.Duration,
) *OutboxRelay {
	return &OutboxRelay{
		store:     store,
		publisher: publisher,
		log:       log,
		interval:  interval,
		batchSize: DefaultOutboxBatchSize,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) error {
	if r.interval <= 0 {
		return fmt.Errorf("%w: polling interval must be positive", ErrInvalidOutboxConfig)
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.PublishPending(ctx); err != nil && ctx.Err() == nil {
			r.log.Error("outbox publish failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelay) PublishPending(ctx context.Context) error {
	messages, err := r.store.ListPending(ctx, r.batchSize)
	if err != nil {
		return fmt.Errorf("read outbox: %w", err)
	}

	for _, m := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.publisher.PublishWithID(ctx, m.Subject, m.Payload, m.ID); err != nil {
			return fmt.Errorf("publish outbox %s: %w", m.ID, err)
		}
		if err := r.store.MarkPublished(ctx, m.ID); err != nil {
			return fmt.Errorf("mark outbox %s published: %w", m.ID, err)
		}
	}
	return nil
}
