package nats

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/contracts/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
)

var errTest = errors.New("test error")

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

func newMsg(t *testing.T, subject string, data []byte) *MockMsg {
	t.Helper()

	msg := NewMockMsg(t)
	msg.EXPECT().Subject().Return(subject).Maybe()
	msg.EXPECT().Data().Return(data).Maybe()
	return msg
}

func expectDeadLetter(msg *MockMsg, js *MockJetStream, subject string, publishErr error) {
	msg.EXPECT().Metadata().Return(&jetstream.MsgMetadata{Stream: "S", Sequence: jetstream.SequencePair{Stream: 1}}, nil).Once()
	js.EXPECT().Publish(mock.Anything, subject, mock.Anything, mock.Anything).Return(&jetstream.PubAck{}, publishErr).Once()
	if publishErr != nil {
		msg.EXPECT().NakWithDelay(5 * time.Second).Return(nil).Once()
		return
	}
	msg.EXPECT().Ack().Return(nil).Once()
}

func TestUserBlocksConsumer_Handle(t *testing.T) {
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	event := contract.BlockEvent{UserID: uuid.NewString(), BlockedID: uuid.NewString(), OccurredAt: at}
	change := func(blocked bool) usecase.BlockChange {
		return usecase.BlockChange{UserID: chat.UserID(event.UserID), BlockedID: chat.UserID(event.BlockedID), Blocked: blocked, At: at}
	}

	tests := []struct {
		name    string
		subject string
		data    []byte
		setup   func(*MockMsg, *MockJetStream, *MockBlockApplier)
	}{
		{
			name:    "applies block",
			subject: contract.SubjectBlocked,
			data:    mustJSON(t, event),
			setup: func(msg *MockMsg, _ *MockJetStream, applier *MockBlockApplier) {
				applier.EXPECT().Execute(mock.Anything, change(true)).Return(nil).Once()
				msg.EXPECT().Ack().Return(nil).Once()
			},
		},
		{
			name:    "applies unblock",
			subject: contract.SubjectUnblocked,
			data:    mustJSON(t, event),
			setup: func(msg *MockMsg, _ *MockJetStream, applier *MockBlockApplier) {
				applier.EXPECT().Execute(mock.Anything, change(false)).Return(nil).Once()
				msg.EXPECT().Ack().Return(nil).Once()
			},
		},
		{
			name:    "retries failures",
			subject: contract.SubjectBlocked,
			data:    mustJSON(t, event),
			setup: func(msg *MockMsg, _ *MockJetStream, applier *MockBlockApplier) {
				applier.EXPECT().Execute(mock.Anything, change(true)).Return(errTest).Once()
				msg.EXPECT().NakWithDelay(5 * time.Second).Return(nil).Once()
			},
		},
		{
			name:    "dead-letters malformed JSON",
			subject: contract.SubjectBlocked,
			data:    []byte(`{`),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockBlockApplier) {
				expectDeadLetter(msg, js, subjectUserBlocksDLQ, nil)
			},
		},
		{
			name:    "dead-letters invalid event",
			subject: contract.SubjectBlocked,
			data:    mustJSON(t, contract.BlockEvent{UserID: event.UserID, BlockedID: event.UserID, OccurredAt: at}),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockBlockApplier) {
				expectDeadLetter(msg, js, subjectUserBlocksDLQ, nil)
			},
		},
		{
			name:    "retries when dead letter fails",
			subject: contract.SubjectBlocked,
			data:    []byte(`{`),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockBlockApplier) {
				expectDeadLetter(msg, js, subjectUserBlocksDLQ, errTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := newMsg(t, tt.subject, tt.data)
			js := NewMockJetStream(t)
			applier := NewMockBlockApplier(t)
			tt.setup(msg, js, applier)

			NewUserBlocksConsumer(nil, js, applier, testLogger()).handle(context.Background(), msg)
		})
	}
}

func TestMessageUpdatesConsumer_Handle(t *testing.T) {
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	edited := message.EditedEvent{MessageID: uuid.NewString(), ChatID: uuid.NewString(), Type: message.TypeText, Payload: json.RawMessage(`{"text":"hi"}`), EditedAt: at}
	deleted := message.DeletedEvent{MessageID: uuid.NewString(), ChatID: uuid.NewString(), DeletedAt: at}
	read := message.ReadEvent{ChatID: uuid.NewString(), UserID: uuid.NewString(), MessageID: uuid.NewString(), ReadAt: at}

	tests := []struct {
		name    string
		subject string
		data    []byte
		setup   func(*MockMsg, *MockJetStream, *MockMessageUpdateDelivery)
	}{
		{
			name:    "delivers edited",
			subject: message.SubjectMessageEdited,
			data:    mustJSON(t, edited),
			setup: func(msg *MockMsg, _ *MockJetStream, d *MockMessageUpdateDelivery) {
				d.EXPECT().DeliverEdited(mock.Anything, mock.MatchedBy(func(e message.EditedEvent) bool {
					return e.MessageID == edited.MessageID && string(e.Payload) == string(edited.Payload)
				})).Return(nil).Once()
				msg.EXPECT().Ack().Return(nil).Once()
			},
		},
		{
			name:    "delivers deleted",
			subject: message.SubjectMessageDeleted,
			data:    mustJSON(t, deleted),
			setup: func(msg *MockMsg, _ *MockJetStream, d *MockMessageUpdateDelivery) {
				d.EXPECT().DeliverDeleted(mock.Anything, deleted).Return(nil).Once()
				msg.EXPECT().Ack().Return(nil).Once()
			},
		},
		{
			name:    "delivers read",
			subject: message.SubjectMessageRead,
			data:    mustJSON(t, read),
			setup: func(msg *MockMsg, _ *MockJetStream, d *MockMessageUpdateDelivery) {
				d.EXPECT().DeliverRead(mock.Anything, read).Return(nil).Once()
				msg.EXPECT().Ack().Return(nil).Once()
			},
		},
		{
			name:    "retries delivery failures",
			subject: message.SubjectMessageDeleted,
			data:    mustJSON(t, deleted),
			setup: func(msg *MockMsg, _ *MockJetStream, d *MockMessageUpdateDelivery) {
				d.EXPECT().DeliverDeleted(mock.Anything, deleted).Return(errTest).Once()
				msg.EXPECT().NakWithDelay(5 * time.Second).Return(nil).Once()
			},
		},
		{
			name:    "dead-letters invalid event",
			subject: message.SubjectMessageDeleted,
			data:    mustJSON(t, message.DeletedEvent{MessageID: "bad"}),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockMessageUpdateDelivery) {
				expectDeadLetter(msg, js, subjectMessageUpdatesDLQ, nil)
			},
		},
		{
			name:    "dead-letters unexpected subject",
			subject: "messages.unknown",
			data:    []byte(`{}`),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockMessageUpdateDelivery) {
				expectDeadLetter(msg, js, subjectMessageUpdatesDLQ, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := newMsg(t, tt.subject, tt.data)
			js := NewMockJetStream(t)
			delivery := NewMockMessageUpdateDelivery(t)
			tt.setup(msg, js, delivery)

			NewMessageUpdatesConsumer(nil, js, delivery, testLogger()).handle(context.Background(), msg)
		})
	}
}

func TestMessageConsumer_Handle(t *testing.T) {
	event := message.CreatedEvent{
		MessageID: uuid.NewString(),
		ChatID:    uuid.NewString(),
		FromUser:  uuid.NewString(),
		Type:      message.TypeText,
		Payload:   json.RawMessage(`{"text":"hi"}`),
		CreatedAt: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
	}
	matchesEvent := mock.MatchedBy(func(e message.CreatedEvent) bool { return e.MessageID == event.MessageID })

	t.Run("delivers and acks", func(t *testing.T) {
		msg := newMsg(t, message.SubjectMessageCreated, mustJSON(t, event))
		delivery := NewMockMessageDelivery(t)
		delivery.EXPECT().Deliver(mock.Anything, matchesEvent).Return(nil).Once()
		msg.EXPECT().Ack().Return(nil).Once()

		NewMessageConsumer(nil, NewMockJetStream(t), delivery, testLogger()).handle(context.Background(), msg)
	})

	t.Run("retries delivery failures", func(t *testing.T) {
		msg := newMsg(t, message.SubjectMessageCreated, mustJSON(t, event))
		delivery := NewMockMessageDelivery(t)
		delivery.EXPECT().Deliver(mock.Anything, matchesEvent).Return(errTest).Once()
		msg.EXPECT().NakWithDelay(5 * time.Second).Return(nil).Once()

		NewMessageConsumer(nil, NewMockJetStream(t), delivery, testLogger()).handle(context.Background(), msg)
	})

	t.Run("dead-letters malformed JSON", func(t *testing.T) {
		msg := newMsg(t, message.SubjectMessageCreated, []byte(`{`))
		js := NewMockJetStream(t)
		expectDeadLetter(msg, js, subjectMessageCreatedDLQ, nil)

		NewMessageConsumer(nil, js, NewMockMessageDelivery(t), testLogger()).handle(context.Background(), msg)
	})
}
