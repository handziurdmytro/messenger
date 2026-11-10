package outbound

import "context"

type OutboxEvent struct {
	ID      string
	Subject string
	Payload []byte
}

type OutboxStore interface {
	ListPending(ctx context.Context, limit int32) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
}
