package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/handziurdmytro/messenger/pkg/errkit"
)

const (
	subjectMessageCreatedDLQ = "chat.messages.created.dlq"
	subjectUserBlocksDLQ     = "chat.user.blocks.dlq"
	subjectMessageUpdatesDLQ = "chat.messages.updates.dlq"
)

var ErrDLQPublishFailed = errors.New("dead letter publish failed")

type deadLetter struct {
	Subject  string    `json:"subject"`
	Data     []byte    `json:"data"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}

type deadLetterPublisher struct {
	js      jetstream.JetStream
	subject string
}

func newDeadLetterPublisher(js jetstream.JetStream, subject string) *deadLetterPublisher {
	return &deadLetterPublisher{js: js, subject: subject}
}

func (p *deadLetterPublisher) Publish(ctx context.Context, msg jetstream.Msg, cause error) error {
	meta, err := msg.Metadata()
	if err != nil {
		return fmt.Errorf("read message metadata: %w", err)
	}

	data, err := json.Marshal(deadLetter{
		Subject:  msg.Subject(),
		Data:     msg.Data(),
		Reason:   cause.Error(),
		FailedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal dead letter: %w", err)
	}

	id := fmt.Sprintf("%s:%d", meta.Stream, meta.Sequence.Stream)
	if _, err := p.js.Publish(ctx, p.subject, data, jetstream.WithMsgID(id)); err != nil {
		return errkit.Wrap(ErrDLQPublishFailed, "publish dead letter", err)
	}

	return nil
}
