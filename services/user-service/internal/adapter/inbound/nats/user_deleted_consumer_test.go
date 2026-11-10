package nats

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	contract "github.com/handziurdmytro/messenger/pkg/contracts/user"
	domainuser "github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440001"

func deletedEventData(t *testing.T, deletedAt time.Time) []byte {
	t.Helper()

	data, err := json.Marshal(contract.DeletedEvent{UserID: testUserID, DeletedAt: deletedAt})
	require.NoError(t, err)
	return data
}

func expectDeadLetter(msg *MockMsg, js *MockJetStream, delivered uint64) {
	msg.EXPECT().Subject().Return(contract.SubjectDeleted).Maybe()
	msg.EXPECT().Metadata().Return(&jetstream.MsgMetadata{
		NumDelivered: delivered,
		Stream:       "USERS",
		Sequence:     jetstream.SequencePair{Stream: 7},
	}, nil).Maybe()
	js.EXPECT().
		Publish(mock.Anything, contract.SubjectDeletedDLQ, mock.Anything, mock.Anything).
		Return(&jetstream.PubAck{}, nil).
		Once()
	msg.EXPECT().DoubleAck(mock.Anything).Return(nil).Once()
}

func TestUserDeletedConsumer_Handle(t *testing.T) {
	deletedAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		data  func(*testing.T) []byte
		setup func(*MockMsg, *MockJetStream, *MockUserDeleter)
	}{
		{
			name: "acks after deletion",
			data: func(t *testing.T) []byte { return deletedEventData(t, deletedAt) },
			setup: func(msg *MockMsg, _ *MockJetStream, deleter *MockUserDeleter) {
				deleter.EXPECT().Execute(mock.Anything, domainuser.ID(testUserID), deletedAt).Return(nil).Once()
				msg.EXPECT().DoubleAck(mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "retries while user is not created yet",
			data: func(t *testing.T) []byte { return deletedEventData(t, deletedAt) },
			setup: func(msg *MockMsg, _ *MockJetStream, deleter *MockUserDeleter) {
				deleter.EXPECT().Execute(mock.Anything, domainuser.ID(testUserID), deletedAt).Return(domainuser.ErrNotFound).Once()
				msg.EXPECT().Metadata().Return(&jetstream.MsgMetadata{NumDelivered: 1}, nil).Once()
				msg.EXPECT().NakWithDelay(retryDelay).Return(nil).Once()
			},
		},
		{
			name: "dead-letters when user never appears",
			data: func(t *testing.T) []byte { return deletedEventData(t, deletedAt) },
			setup: func(msg *MockMsg, js *MockJetStream, deleter *MockUserDeleter) {
				deleter.EXPECT().Execute(mock.Anything, domainuser.ID(testUserID), deletedAt).Return(domainuser.ErrNotFound).Once()
				expectDeadLetter(msg, js, maxNotFoundDeliveries)
			},
		},
		{
			name: "dead-letters invalid timestamp",
			data: func(t *testing.T) []byte { return deletedEventData(t, deletedAt) },
			setup: func(msg *MockMsg, js *MockJetStream, deleter *MockUserDeleter) {
				deleter.EXPECT().Execute(mock.Anything, domainuser.ID(testUserID), deletedAt).Return(domainuser.ErrInvalidTimestamp).Once()
				expectDeadLetter(msg, js, 1)
			},
		},
		{
			name: "retries unexpected errors",
			data: func(t *testing.T) []byte { return deletedEventData(t, deletedAt) },
			setup: func(msg *MockMsg, _ *MockJetStream, deleter *MockUserDeleter) {
				deleter.EXPECT().Execute(mock.Anything, domainuser.ID(testUserID), deletedAt).Return(errors.New("db down")).Once()
				msg.EXPECT().NakWithDelay(retryDelay).Return(nil).Once()
			},
		},
		{
			name: "dead-letters malformed JSON",
			data: func(*testing.T) []byte { return []byte(`{`) },
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockUserDeleter) {
				expectDeadLetter(msg, js, 1)
			},
		},
		{
			name: "dead-letters invalid event",
			data: func(t *testing.T) []byte { return deletedEventData(t, time.Time{}) },
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockUserDeleter) {
				expectDeadLetter(msg, js, 1)
			},
		},
		{
			name: "retries when dead letter publish fails",
			data: func(*testing.T) []byte { return []byte(`{`) },
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockUserDeleter) {
				msg.EXPECT().Subject().Return(contract.SubjectDeleted).Maybe()
				msg.EXPECT().Metadata().Return(&jetstream.MsgMetadata{Stream: "USERS"}, nil).Once()
				js.EXPECT().Publish(mock.Anything, contract.SubjectDeletedDLQ, mock.Anything, mock.Anything).Return(nil, errors.New("nats down")).Once()
				msg.EXPECT().NakWithDelay(retryDelay).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMockMsg(t)
			js := NewMockJetStream(t)
			deleter := NewMockUserDeleter(t)
			msg.EXPECT().Data().Return(tt.data(t)).Maybe()
			tt.setup(msg, js, deleter)

			consumer := NewUserDeletedConsumer(nil, js, deleter, slog.New(slog.NewTextHandler(io.Discard, nil)))
			consumer.handle(context.Background(), msg)
		})
	}
}
