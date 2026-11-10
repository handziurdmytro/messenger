package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type BlockApplier interface {
	Execute(ctx context.Context, change usecase.BlockChange) error
}

type UserBlocksConsumer struct {
	consumer jetstream.Consumer
	applier  BlockApplier
	dlq      *deadLetterPublisher
	logger   *slog.Logger
}

func NewUserBlocksConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	applier BlockApplier,
	logger *slog.Logger,
) *UserBlocksConsumer {
	return &UserBlocksConsumer{
		consumer: consumer,
		applier:  applier,
		dlq:      newDeadLetterPublisher(js, subjectUserBlocksDLQ),
		logger:   logger,
	}
}

func (c *UserBlocksConsumer) Run(ctx context.Context) error {
	cc, err := c.consumer.Consume(
		func(msg jetstream.Msg) {
			c.handle(ctx, msg)
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			c.logger.Error("user blocks consumer error", "error", err)
		}),
	)
	if err != nil {
		return errkit.Wrap(ErrConsumeFailed, "start user blocks consumer", err)
	}

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (c *UserBlocksConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var event contract.BlockEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.reject(ctx, msg, err)
		return
	}
	if err := event.Validate(); err != nil {
		c.reject(ctx, msg, err)
		return
	}

	change := usecase.BlockChange{
		UserID:    chat.UserID(event.UserID),
		BlockedID: chat.UserID(event.BlockedID),
		Blocked:   msg.Subject() == contract.SubjectBlocked,
		At:        event.OccurredAt,
	}

	if err := c.applier.Execute(ctx, change); err != nil {
		c.logger.Error("apply user block failed", "user_id", event.UserID, "error", err)
		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak user block failed", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack user block failed", "error", err)
	}
}

func (c *UserBlocksConsumer) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	c.logger.Error("invalid user block event", "subject", msg.Subject(), "error", cause)

	if err := c.dlq.Publish(ctx, msg, cause); err != nil {
		c.logger.Error("dead letter publish failed", "error", err)
		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak user block failed", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack dead-lettered user block failed", "error", err)
	}
}
