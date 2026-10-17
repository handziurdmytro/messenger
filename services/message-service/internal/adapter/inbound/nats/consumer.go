package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/nats-io/nats.go/jetstream"
)

const retryDelay = 5 * time.Second

var ErrConsumeFailed = errors.New("message consumption failed")

type MessageConsumer struct {
	consumer jetstream.Consumer
	send     *usecase.SendMessage
	dlq      *deadLetterPublisher
	logger   *slog.Logger
}

func NewMessageConsumer(
	consumer jetstream.Consumer,
	js jetstream.JetStream,
	send *usecase.SendMessage,
	logger *slog.Logger,
) *MessageConsumer {
	return &MessageConsumer{
		consumer: consumer,
		send:     send,
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

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (c *MessageConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	var cmd message.SendMessageCommand

	if err := json.Unmarshal(msg.Data(), &cmd); err != nil {
		c.reject(ctx, msg, err)
		return
	}
	if err := cmd.Validate(); err != nil {
		c.reject(ctx, msg, err)
		return
	}

	if err := c.send.Execute(ctx, cmd); err != nil {
		c.logger.Error("send message failed", "message_id", cmd.MessageID, "error", err)
		c.retry(msg)
		return
	}

	if err := msg.DoubleAck(ctx); err != nil {
		c.logger.Error("ack message failed", "message_id", cmd.MessageID, "error", err)
	}
}

func (c *MessageConsumer) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	c.logger.Error("invalid send message command", "subject", msg.Subject(), "error", cause)

	if err := c.dlq.Publish(ctx, msg, cause); err != nil {
		c.logger.Error("dead letter publish failed", "error", err)
		c.retry(msg)
		return
	}

	if err := msg.DoubleAck(ctx); err != nil {
		c.logger.Error("ack dead-lettered message failed", "error", err)
	}
}

func (c *MessageConsumer) retry(msg jetstream.Msg) {
	if err := msg.NakWithDelay(retryDelay); err != nil {
		c.logger.Error("nak message failed", "error", err)
	}
}
