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

func TestUserCreatedConsumer_Handle(t *testing.T) {
	createdAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	valid, err := json.Marshal(contract.CreatedEvent{UserID: testUserID, Email: "alice@example.com", CreatedAt: createdAt})
	require.NoError(t, err)

	savedUser := mock.MatchedBy(func(u domainuser.User) bool {
		return u.ID() == testUserID && u.Email().String() == "alice@example.com" && u.CreatedAt().Equal(createdAt)
	})

	deadLetter := func(msg *MockMsg, js *MockJetStream) {
		msg.EXPECT().Subject().Return(contract.SubjectCreated).Maybe()
		msg.EXPECT().Metadata().Return(&jetstream.MsgMetadata{Stream: "USERS"}, nil).Once()
		js.EXPECT().Publish(mock.Anything, contract.SubjectCreatedDLQ, mock.Anything, mock.Anything).Return(&jetstream.PubAck{}, nil).Once()
		msg.EXPECT().DoubleAck(mock.Anything).Return(nil).Once()
	}

	tests := []struct {
		name  string
		data  []byte
		setup func(*MockMsg, *MockJetStream, *MockUserRepository)
	}{
		{
			name: "saves user and acks",
			data: valid,
			setup: func(msg *MockMsg, _ *MockJetStream, users *MockUserRepository) {
				users.EXPECT().Save(mock.Anything, savedUser).Return(nil).Once()
				msg.EXPECT().DoubleAck(mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "dead-letters duplicate user",
			data: valid,
			setup: func(msg *MockMsg, js *MockJetStream, users *MockUserRepository) {
				users.EXPECT().Save(mock.Anything, savedUser).Return(domainuser.ErrAlreadyExists).Once()
				deadLetter(msg, js)
			},
		},
		{
			name: "dead-letters taken email",
			data: valid,
			setup: func(msg *MockMsg, js *MockJetStream, users *MockUserRepository) {
				users.EXPECT().Save(mock.Anything, savedUser).Return(domainuser.ErrEmailTaken).Once()
				deadLetter(msg, js)
			},
		},
		{
			name: "retries persistence errors",
			data: valid,
			setup: func(msg *MockMsg, _ *MockJetStream, users *MockUserRepository) {
				users.EXPECT().Save(mock.Anything, savedUser).Return(errors.New("db down")).Once()
				msg.EXPECT().NakWithDelay(retryDelay).Return(nil).Once()
			},
		},
		{
			name: "dead-letters malformed JSON",
			data: []byte(`{`),
			setup: func(msg *MockMsg, js *MockJetStream, _ *MockUserRepository) {
				deadLetter(msg, js)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMockMsg(t)
			js := NewMockJetStream(t)
			users := NewMockUserRepository(t)
			msg.EXPECT().Data().Return(tt.data).Maybe()
			tt.setup(msg, js, users)

			consumer := NewUserCreatedConsumer(nil, js, users, slog.New(slog.NewTextHandler(io.Discard, nil)))
			consumer.handle(context.Background(), msg)
		})
	}
}
