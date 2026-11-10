package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	domainuser "github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type UserCreatedConsumer struct {
	consumer jetstream.Consumer
	delivery delivery
	users    outbound.UserRepository
	log      *slog.Logger
}

func NewUserCreatedConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	users outbound.UserRepository,
	log *slog.Logger,
) *UserCreatedConsumer {
	return &UserCreatedConsumer{
		consumer: consumer,
		delivery: delivery{js: js, dlqSubject: contract.SubjectCreatedDLQ, log: log},
		users:    users,
		log:      log,
	}
}

func (c *UserCreatedConsumer) Run(ctx context.Context) error {
	return consume(ctx, c.consumer, "user created", c.log, c.handle)
}

func (c *UserCreatedConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var event contract.CreatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.delivery.reject(ctx, msg, err)
		return
	}
	if err := event.Validate(); err != nil {
		c.delivery.reject(ctx, msg, err)
		return
	}

	u, err := domainuser.New(
		domainuser.ID(event.UserID),
		event.Email,
		event.CreatedAt,
	)
	if err != nil {
		c.delivery.reject(ctx, msg, err)
		return
	}

	if err := c.users.Save(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrAlreadyExists) || errors.Is(err, domainuser.ErrEmailTaken) {
			c.delivery.reject(ctx, msg, err)
			return
		}
		c.log.Error("save user failed", "user_id", event.UserID, "error", err)
		c.delivery.retry(msg)
		return
	}

	c.delivery.ack(ctx, msg)
}
