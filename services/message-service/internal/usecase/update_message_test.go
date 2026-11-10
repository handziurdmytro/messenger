package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
)

var errTest = errors.New("test error")

func newTestMessage(t *testing.T, author string, msgType message.Type, payload string) message.Message {
	t.Helper()

	createdAt := time.Now().UTC().Add(-time.Minute)
	msg, err := message.Restore(
		message.ID(uuid.NewString()),
		message.ChatID(uuid.NewString()),
		author,
		msgType,
		json.RawMessage(payload),
		createdAt,
		nil,
		nil,
	)
	require.NoError(t, err)
	return msg
}

func TestEditMessage_Execute(t *testing.T) {
	author := uuid.NewString()
	newPayload := json.RawMessage(`{"text":"edited"}`)

	tests := []struct {
		name    string
		userID  string
		payload json.RawMessage
		msgType message.Type
		findErr error
		setup   func(*MockMessageRepository, *MockMessagePublisher, message.Message)
		wantErr error
	}{
		{
			name:    "edits and publishes",
			userID:  author,
			payload: newPayload,
			setup: func(repo *MockMessageRepository, pub *MockMessagePublisher, msg message.Message) {
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(m message.Message) bool {
					return m.ID == msg.ID && string(m.Payload) == string(newPayload) && m.EditedAt != nil
				})).Return(nil).Once()
				pub.EXPECT().PublishEdited(mock.Anything, mock.MatchedBy(func(e contract.EditedEvent) bool {
					return e.MessageID == string(msg.ID) && string(e.Payload) == string(newPayload) && !e.EditedAt.IsZero()
				})).Return(nil).Once()
			},
		},
		{
			name:    "not the author",
			userID:  uuid.NewString(),
			payload: newPayload,
			wantErr: ErrForbidden,
		},
		{
			name:    "message not found",
			userID:  author,
			payload: newPayload,
			findErr: ErrMessageNotFound,
			wantErr: ErrMessageNotFound,
		},
		{
			name:    "invalid payload",
			userID:  author,
			payload: json.RawMessage(`{"text":""}`),
			wantErr: contract.ErrInvalidPayload,
		},
		{
			name:    "system messages are not editable",
			userID:  author,
			payload: newPayload,
			msgType: message.TypeSystem,
			wantErr: message.ErrMessageNotEditable,
		},
		{
			name:    "update fails",
			userID:  author,
			payload: newPayload,
			setup: func(repo *MockMessageRepository, _ *MockMessagePublisher, _ message.Message) {
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(errTest).Once()
			},
			wantErr: ErrPersistence,
		},
		{
			name:    "publish fails",
			userID:  author,
			payload: newPayload,
			setup: func(repo *MockMessageRepository, pub *MockMessagePublisher, _ message.Message) {
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Once()
				pub.EXPECT().PublishEdited(mock.Anything, mock.Anything).Return(errTest).Once()
			},
			wantErr: ErrPublishFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgType := tt.msgType
			if msgType == "" {
				msgType = message.TypeText
			}
			msg := newTestMessage(t, author, msgType, `{"text":"original"}`)

			repo := NewMockMessageRepository(t)
			pub := NewMockMessagePublisher(t)
			repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, tt.findErr).Once()
			if tt.setup != nil {
				tt.setup(repo, pub, msg)
			}

			got, err := NewEditMessage(repo, pub).Execute(context.Background(), EditMessageInput{
				ChatID: msg.ChatID, MessageID: msg.ID, UserID: tt.userID, Payload: tt.payload,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, string(newPayload), string(got.Payload))
		})
	}

	t.Run("invalid IDs", func(t *testing.T) {
		uc := NewEditMessage(NewMockMessageRepository(t), NewMockMessagePublisher(t))

		_, err := uc.Execute(context.Background(), EditMessageInput{ChatID: "bad", MessageID: message.ID(uuid.NewString())})
		require.ErrorIs(t, err, ErrInvalidChatID)
		_, err = uc.Execute(context.Background(), EditMessageInput{ChatID: message.ChatID(uuid.NewString()), MessageID: "bad"})
		require.ErrorIs(t, err, ErrInvalidMessageID)
	})
}

func TestDeleteMessage_Execute(t *testing.T) {
	author := uuid.NewString()

	tests := []struct {
		name    string
		userID  string
		deleted bool
		setup   func(*MockMessageRepository, *MockMessagePublisher, message.Message)
		wantErr error
	}{
		{
			name:   "deletes and publishes",
			userID: author,
			setup: func(repo *MockMessageRepository, pub *MockMessagePublisher, msg message.Message) {
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(m message.Message) bool {
					return m.ID == msg.ID && m.DeletedAt != nil && m.Payload == nil
				})).Return(nil).Once()
				pub.EXPECT().PublishDeleted(mock.Anything, mock.MatchedBy(func(e contract.DeletedEvent) bool {
					return e.MessageID == string(msg.ID) && e.ChatID == string(msg.ChatID)
				})).Return(nil).Once()
			},
		},
		{
			name:    "not the author",
			userID:  uuid.NewString(),
			wantErr: ErrForbidden,
		},
		{
			name:    "already deleted",
			userID:  author,
			deleted: true,
			wantErr: message.ErrMessageDeleted,
		},
		{
			name:   "publish fails",
			userID: author,
			setup: func(repo *MockMessageRepository, pub *MockMessagePublisher, _ message.Message) {
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Once()
				pub.EXPECT().PublishDeleted(mock.Anything, mock.Anything).Return(errTest).Once()
			},
			wantErr: ErrPublishFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := newTestMessage(t, author, message.TypeText, `{"text":"original"}`)
			if tt.deleted {
				require.NoError(t, msg.Delete(time.Now().UTC()))
			}

			repo := NewMockMessageRepository(t)
			pub := NewMockMessagePublisher(t)
			repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()
			if tt.setup != nil {
				tt.setup(repo, pub, msg)
			}

			err := NewDeleteMessage(repo, pub).Execute(context.Background(), msg.ChatID, msg.ID, tt.userID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
