package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const retryDelay = 5 * time.Second

type delivery struct {
	js         jetstream.JetStream
	dlqSubject string
	log        *slog.Logger
}

func consume(ctx context.Context, consumer jetstream.Consumer, name string, log *slog.Logger, handle func(context.Context, jetstream.Msg)) error {
	cc, err := consumer.Consume(
		func(msg jetstream.Msg) { handle(ctx, msg) },
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			log.Error(name+" consumer error", "error", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("start %s consumer: %w", name, err)
	}

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (d delivery) ack(ctx context.Context, msg jetstream.Msg) {
	if err := msg.DoubleAck(ctx); err != nil {
		d.log.Error("ack event failed", "subject", msg.Subject(), "error", err)
	}
}

func (d delivery) reject(ctx context.Context, msg jetstream.Msg, cause error) {
	d.log.Error("reject event", "subject", msg.Subject(), "error", cause)

	meta, err := msg.Metadata()
	if err != nil {
		d.retry(msg)
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
		d.retry(msg)
		return
	}

	id := fmt.Sprintf("%s:%d", meta.Stream, meta.Sequence.Stream)
	if _, err := d.js.Publish(ctx, d.dlqSubject, body, jetstream.WithMsgID(id)); err != nil {
		d.log.Error("publish DLQ failed", "subject", d.dlqSubject, "error", err)
		d.retry(msg)
		return
	}

	d.ack(ctx, msg)
}

func (d delivery) retry(msg jetstream.Msg) {
	if err := msg.NakWithDelay(retryDelay); err != nil {
		d.log.Error("nak event failed", "subject", msg.Subject(), "error", err)
	}
}
