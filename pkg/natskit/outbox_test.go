package natskit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestOutboxRelay_PublishPending(t *testing.T) {
	first := OutboxMessage{ID: "1", Subject: "user.created", Payload: []byte(`{"a":1}`)}
	second := OutboxMessage{ID: "2", Subject: "user.deleted", Payload: []byte(`{"b":2}`)}

	tests := []struct {
		name    string
		setup   func(*MockOutboxStore, *MockMessagePublisher)
		wantErr error
	}{
		{
			name: "publishes and marks every pending message",
			setup: func(store *MockOutboxStore, pub *MockMessagePublisher) {
				store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return([]OutboxMessage{first, second}, nil).Once()
				pub.EXPECT().PublishWithID(mock.Anything, first.Subject, first.Payload, first.ID).Return(nil).Once()
				store.EXPECT().MarkPublished(mock.Anything, first.ID).Return(nil).Once()
				pub.EXPECT().PublishWithID(mock.Anything, second.Subject, second.Payload, second.ID).Return(nil).Once()
				store.EXPECT().MarkPublished(mock.Anything, second.ID).Return(nil).Once()
			},
		},
		{
			name: "nothing pending",
			setup: func(store *MockOutboxStore, _ *MockMessagePublisher) {
				store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return(nil, nil).Once()
			},
		},
		{
			name: "list fails",
			setup: func(store *MockOutboxStore, _ *MockMessagePublisher) {
				store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return(nil, errTest).Once()
			},
			wantErr: errTest,
		},
		{
			name: "publish fails and stops before marking",
			setup: func(store *MockOutboxStore, pub *MockMessagePublisher) {
				store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return([]OutboxMessage{first, second}, nil).Once()
				pub.EXPECT().PublishWithID(mock.Anything, first.Subject, first.Payload, first.ID).Return(errTest).Once()
			},
			wantErr: errTest,
		},
		{
			name: "mark fails and stops the batch",
			setup: func(store *MockOutboxStore, pub *MockMessagePublisher) {
				store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return([]OutboxMessage{first, second}, nil).Once()
				pub.EXPECT().PublishWithID(mock.Anything, first.Subject, first.Payload, first.ID).Return(nil).Once()
				store.EXPECT().MarkPublished(mock.Anything, first.ID).Return(errTest).Once()
			},
			wantErr: errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockOutboxStore(t)
			pub := NewMockMessagePublisher(t)
			tt.setup(store, pub)

			err := NewOutboxRelay(store, pub, testLogger(), time.Second).PublishPending(context.Background())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOutboxRelay_PublishPendingStopsOnCanceledContext(t *testing.T) {
	store := NewMockOutboxStore(t)
	pub := NewMockMessagePublisher(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).Return([]OutboxMessage{{ID: "1", Subject: "s"}}, nil).Once()

	err := NewOutboxRelay(store, pub, testLogger(), time.Second).PublishPending(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutboxRelay_Run(t *testing.T) {
	t.Run("rejects non-positive interval", func(t *testing.T) {
		err := NewOutboxRelay(NewMockOutboxStore(t), NewMockMessagePublisher(t), testLogger(), 0).Run(context.Background())
		require.ErrorIs(t, err, ErrInvalidOutboxConfig)
	})

	t.Run("polls until context is canceled", func(t *testing.T) {
		store := NewMockOutboxStore(t)
		ctx, cancel := context.WithCancel(context.Background())

		calls := 0
		store.EXPECT().ListPending(mock.Anything, DefaultOutboxBatchSize).RunAndReturn(func(context.Context, int32) ([]OutboxMessage, error) {
			calls++
			if calls == 2 {
				cancel()
			}
			return nil, errTest
		})

		err := NewOutboxRelay(store, NewMockMessagePublisher(t), testLogger(), time.Millisecond).Run(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, calls, 2)
	})
}
