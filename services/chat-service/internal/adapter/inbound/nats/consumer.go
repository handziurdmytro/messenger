package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/nats-io/nats.go/jetstream"
)

const consumerDrainTimeout = 3 * time.Second

var ErrConsumeFailed = errors.New("message consumption failed")

type MessageDelivery interface {
	Deliver(ctx context.Context, event message.CreatedEvent) error
}

type MessageConsumer struct {
	consumer jetstream.Consumer
	delivery MessageDelivery
	dlq      *deadLetterPublisher
	logger   *slog.Logger
}

func NewMessageConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	delivery MessageDelivery,
	logger *slog.Logger,
) *MessageConsumer {
	return &MessageConsumer{
		consumer: consumer,
		delivery: delivery,
		dlq:      newDeadLetterPublisher(js),
		logger:   logger,
	}
}

func (c *MessageConsumer) Run(ctx context.Context) error {
	cc, err := c.consumer.Consume(
		func(msg jetstream.Msg) {
			c.handle(ctx, msg)
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			c.logger.Error("nats consumer error", "error", err)
		}),
	)
	if err != nil {
		return errkit.Wrap(ErrConsumeFailed, "start message consumer", err)
	}
	defer cc.Stop()

	<-ctx.Done()

	cc.Drain()

	timer := time.NewTimer(consumerDrainTimeout)
	defer timer.Stop()

	select {
	case <-cc.Closed():
	case <-timer.C:
		c.logger.Warn("nats consumer drain timed out")
		cc.Stop()
	}

	return nil
}

func (c *MessageConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var event message.CreatedEvent

	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.reject(ctx, msg, err)
		return
	}

	if err := event.Validate(); err != nil {
		c.reject(ctx, msg, err)
		return
	}

	if err := c.delivery.Deliver(ctx, event); err != nil {
		c.logger.Error("message delivery failed",
			"message_id", event.MessageID,
			"error", err,
		)

		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak message failed", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack message failed",
			"message_id", event.MessageID,
			"error", err,
		)
	}
}

func (c *MessageConsumer) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	c.logger.Error("invalid message event",
		"subject", msg.Subject(),
		"error", cause,
	)

	if err := c.dlq.Publish(ctx, msg, cause); err != nil {
		c.logger.Error("dead letter publish failed", "error", err)

		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak message failed", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack dead-lettered message failed", "error", err)
	}
}
