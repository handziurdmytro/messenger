package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/nats-io/nats.go/jetstream"
)

const subjectSendMessageDLQ = "message.messages.send.dlq"

var ErrDLQPublishFailed = errors.New("dead letter publish failed")

type deadLetter struct {
	Subject  string    `json:"subject"`
	Data     []byte    `json:"data"`
	Reason   string    `json:"reason"`
	FailedAt time.Time `json:"failed_at"`
}

type deadLetterPublisher struct {
	js jetstream.JetStream
}

func newDeadLetterPublisher(js jetstream.JetStream) *deadLetterPublisher {
	return &deadLetterPublisher{js: js}
}

func (p *deadLetterPublisher) Publish(ctx context.Context, msg jetstream.Msg, cause error) error {
	meta, err := msg.Metadata()
	if err != nil {
		return errkit.Wrap(ErrDLQPublishFailed, "read message metadata", err)
	}

	data, err := json.Marshal(deadLetter{
		Subject:  msg.Subject(),
		Data:     msg.Data(),
		Reason:   cause.Error(),
		FailedAt: time.Now().UTC(),
	})
	if err != nil {
		return errkit.Wrap(ErrDLQPublishFailed, "marshal dead letter", err)
	}

	id := fmt.Sprintf("%s:%d", meta.Stream, meta.Sequence.Stream)
	if _, err := p.js.Publish(ctx, subjectSendMessageDLQ, data, jetstream.WithMsgID(id)); err != nil {
		return errkit.Wrap(ErrDLQPublishFailed, "publish dead letter", err)
	}
	return nil
}
