package natskit

import (
	"context"
	"strings"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/nats-io/nats.go/jetstream"
)

type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) Publish(ctx context.Context, subject string, data []byte) error {
	return p.publish(ctx, subject, data)
}

func (p *Publisher) PublishWithID(ctx context.Context, subject string, data []byte, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidMessageID
	}

	return p.publish(ctx, subject, data, jetstream.WithMsgID(id))
}

func (p *Publisher) publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	opts = append([]jetstream.PublishOpt{jetstream.WithRetryAttempts(0)}, opts...)

	_, err := p.js.Publish(ctx, subject, data, opts...)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errkit.Wrap(ErrPublishFailed, "natskit: publish", err)
	}

	return nil
}
