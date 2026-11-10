package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	domainuser "github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

const maxNotFoundDeliveries = 10

type UserDeleter interface {
	Execute(ctx context.Context, id domainuser.ID, deletedAt time.Time) error
}

type UserDeletedConsumer struct {
	consumer jetstream.Consumer
	delivery delivery
	deleter  UserDeleter
	log      *slog.Logger
}

func NewUserDeletedConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	deleter UserDeleter,
	log *slog.Logger,
) *UserDeletedConsumer {
	return &UserDeletedConsumer{
		consumer: consumer,
		delivery: delivery{js: js, dlqSubject: contract.SubjectDeletedDLQ, log: log},
		deleter:  deleter,
		log:      log,
	}
}

func (c *UserDeletedConsumer) Run(ctx context.Context) error {
	return consume(ctx, c.consumer, "user deleted", c.log, c.handle)
}

func (c *UserDeletedConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var event contract.DeletedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.delivery.reject(ctx, msg, err)
		return
	}
	if err := event.Validate(); err != nil {
		c.delivery.reject(ctx, msg, err)
		return
	}

	err := c.deleter.Execute(ctx, domainuser.ID(event.UserID), event.DeletedAt)
	switch {
	case err == nil:
		c.delivery.ack(ctx, msg)
	case errors.Is(err, domainuser.ErrNotFound):
		if delivered(msg) >= maxNotFoundDeliveries {
			c.delivery.reject(ctx, msg, err)
			return
		}
		c.log.Warn("deleted user not found yet", "user_id", event.UserID)
		c.delivery.retry(msg)
	case errors.Is(err, domainuser.ErrInvalidTimestamp):
		c.delivery.reject(ctx, msg, err)
	default:
		c.log.Error("delete user failed", "user_id", event.UserID, "error", err)
		c.delivery.retry(msg)
	}
}

func delivered(msg jetstream.Msg) uint64 {
	meta, err := msg.Metadata()
	if err != nil {
		return 0
	}
	return meta.NumDelivered
}
