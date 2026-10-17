package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	domainuser "github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const retryDelay = 5 * time.Second

type UserCreatedConsumer struct {
	consumer jetstream.Consumer
	js       jetstream.JetStream
	users    outbound.UserRepository
	log      *slog.Logger
}

func NewUserCreatedConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	users outbound.UserRepository,
	log *slog.Logger,
) *UserCreatedConsumer {
	return &UserCreatedConsumer{consumer: consumer, js: js, users: users, log: log}
}

func (c *UserCreatedConsumer) Run(ctx context.Context) error {
	cc, err := c.consumer.Consume(
		func(msg jetstream.Msg) { c.handle(ctx, msg) },
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			c.log.Error("user consumer error", "error", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("start user consumer: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (c *UserCreatedConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var event contract.CreatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.reject(ctx, msg, err)
		return
	}
	if err := event.Validate(); err != nil {
		c.reject(ctx, msg, err)
		return
	}

	u, err := domainuser.New(
		domainuser.ID(event.UserID),
		event.Email,
		event.CreatedAt,
	)
	if err != nil {
		c.reject(ctx, msg, err)
		return
	}

	if err := c.users.Save(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrAlreadyExists) || errors.Is(err, domainuser.ErrEmailTaken) {
			c.reject(ctx, msg, err)
			return
		}
		c.log.Error("save user failed", "user_id", event.UserID, "error", err)
		c.retry(msg)
		return
	}

	if err := msg.DoubleAck(ctx); err != nil {
		c.log.Error("ack user event failed", "user_id", event.UserID, "error", err)
	}
}

func (c *UserCreatedConsumer) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	c.log.Error("reject user event", "subject", msg.Subject(), "error", cause)

	meta, err := msg.Metadata()
	if err != nil {
		c.retry(msg)
		return
	}

	body, err := json.Marshal(struct {
		Subject string `json:"subject"`
		Data    []byte `json:"data"`
		Reason  string `json:"reason"`
	}{
		Subject: msg.Subject(),
		Data:    msg.Data(),
		Reason:  cause.Error(),
	})
	if err != nil {
		c.retry(msg)
		return
	}

	id := fmt.Sprintf("%s:%d", meta.Stream, meta.Sequence.Stream)
	if _, err := c.js.Publish(ctx, contract.SubjectCreatedDLQ, body, jetstream.WithMsgID(id)); err != nil {
		c.log.Error("publish user DLQ failed", "error", err)
		c.retry(msg)
		return
	}

	if err := msg.DoubleAck(ctx); err != nil {
		c.log.Error("ack dead-lettered user event failed", "error", err)
	}
}

func (c *UserCreatedConsumer) retry(msg jetstream.Msg) {
	if err := msg.NakWithDelay(retryDelay); err != nil {
		c.log.Error("nak user event failed", "error", err)
	}
}
