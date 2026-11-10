package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var errInvalidEvent = errors.New("invalid event")

type validatable interface {
	Validate() error
}

func decodeEvent[T validatable](data []byte) (T, error) {
	var event T
	if err := json.Unmarshal(data, &event); err != nil {
		return event, fmt.Errorf("%w: %w", errInvalidEvent, err)
	}
	if err := event.Validate(); err != nil {
		return event, fmt.Errorf("%w: %w", errInvalidEvent, err)
	}
	return event, nil
}

type MessageUpdateDelivery interface {
	DeliverEdited(ctx context.Context, event message.EditedEvent) error
	DeliverDeleted(ctx context.Context, event message.DeletedEvent) error
	DeliverRead(ctx context.Context, event message.ReadEvent) error
}

type MessageUpdatesConsumer struct {
	consumer jetstream.Consumer
	delivery MessageUpdateDelivery
	dlq      *deadLetterPublisher
	logger   *slog.Logger
}

func NewMessageUpdatesConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	delivery MessageUpdateDelivery,
	logger *slog.Logger,
) *MessageUpdatesConsumer {
	return &MessageUpdatesConsumer{
		consumer: consumer,
		delivery: delivery,
		dlq:      newDeadLetterPublisher(js, subjectMessageUpdatesDLQ),
		logger:   logger,
	}
}

func (c *MessageUpdatesConsumer) Run(ctx context.Context) error {
	cc, err := c.consumer.Consume(
		func(msg jetstream.Msg) {
			c.handle(ctx, msg)
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			c.logger.Error("message updates consumer error", "error", err)
		}),
	)
	if err != nil {
		return errkit.Wrap(ErrConsumeFailed, "start message updates consumer", err)
	}

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (c *MessageUpdatesConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var err error
	switch msg.Subject() {
	case message.SubjectMessageEdited:
		var event message.EditedEvent
		if event, err = decodeEvent[message.EditedEvent](msg.Data()); err == nil {
			err = c.delivery.DeliverEdited(ctx, event)
		}
	case message.SubjectMessageDeleted:
		var event message.DeletedEvent
		if event, err = decodeEvent[message.DeletedEvent](msg.Data()); err == nil {
			err = c.delivery.DeliverDeleted(ctx, event)
		}
	case message.SubjectMessageRead:
		var event message.ReadEvent
		if event, err = decodeEvent[message.ReadEvent](msg.Data()); err == nil {
			err = c.delivery.DeliverRead(ctx, event)
		}
	default:
		err = fmt.Errorf("%w: unexpected subject %s", errInvalidEvent, msg.Subject())
	}

	switch {
	case err == nil:
		if err := msg.Ack(); err != nil {
			c.logger.Error("ack message update failed", "error", err)
		}
	case errors.Is(err, errInvalidEvent):
		c.reject(ctx, msg, err)
	default:
		c.logger.Error("message update delivery failed", "subject", msg.Subject(), "error", err)
		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak message update failed", "error", err)
		}
	}
}

func (c *MessageUpdatesConsumer) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	c.logger.Error("invalid message update event", "subject", msg.Subject(), "error", cause)

	if err := c.dlq.Publish(ctx, msg, cause); err != nil {
		c.logger.Error("dead letter publish failed", "error", err)
		if err := msg.NakWithDelay(5 * time.Second); err != nil {
			c.logger.Error("nak message update failed", "error", err)
		}
		return
	}

	if err := msg.Ack(); err != nil {
		c.logger.Error("ack dead-lettered message update failed", "error", err)
	}
}
