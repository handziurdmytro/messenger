package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
)

func TestSendMessage_Execute(t *testing.T) {
	cmd := contract.SendMessageCommand{
		MessageID: uuid.NewString(),
		ChatID:    uuid.NewString(),
		FromUser:  uuid.NewString(),
		Type:      contract.TypeText,
		Payload:   json.RawMessage(`{"text":"hi"}`),
	}
	savedAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	saved := func(_ context.Context, msg message.Message) (message.Message, error) {
		msg.CreatedAt = savedAt
		return msg, nil
	}

	t.Run("saves and publishes created event", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		pub := NewMockMessagePublisher(t)
		repo.EXPECT().Save(mock.Anything, mock.AnythingOfType("message.Message")).RunAndReturn(saved).Once()
		pub.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(e contract.CreatedEvent) bool {
			return e.MessageID == cmd.MessageID && e.ChatID == cmd.ChatID && e.CreatedAt.Equal(savedAt)
		})).Return(nil).Once()

		require.NoError(t, NewSendMessage(repo, pub).Execute(context.Background(), cmd))
	})

	t.Run("invalid command", func(t *testing.T) {
		invalid := cmd
		invalid.MessageID = "bad"
		err := NewSendMessage(NewMockMessageRepository(t), NewMockMessagePublisher(t)).Execute(context.Background(), invalid)
		require.ErrorIs(t, err, contract.ErrInvalidMessageID)
	})

	t.Run("save fails", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		repo.EXPECT().Save(mock.Anything, mock.Anything).Return(message.Message{}, errTest).Once()

		err := NewSendMessage(repo, NewMockMessagePublisher(t)).Execute(context.Background(), cmd)
		require.ErrorIs(t, err, ErrPersistence)
	})

	t.Run("publish fails", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		pub := NewMockMessagePublisher(t)
		repo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(saved).Once()
		pub.EXPECT().Publish(mock.Anything, mock.Anything).Return(errTest).Once()

		err := NewSendMessage(repo, pub).Execute(context.Background(), cmd)
		require.ErrorIs(t, err, ErrPublishFailed)
	})
}

func TestGetMessage_Execute(t *testing.T) {
	msg := newTestMessage(t, uuid.NewString(), message.TypeText, `{"text":"hi"}`)

	repo := NewMockMessageRepository(t)
	repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()

	uc := NewGetMessage(repo)
	got, err := uc.Execute(context.Background(), msg.ChatID, msg.ID)
	require.NoError(t, err)
	require.Equal(t, msg.ID, got.ID)

	_, err = uc.Execute(context.Background(), "bad", msg.ID)
	require.ErrorIs(t, err, ErrInvalidChatID)
	_, err = uc.Execute(context.Background(), msg.ChatID, "bad")
	require.ErrorIs(t, err, ErrInvalidMessageID)
}

func TestGetHistory_Execute(t *testing.T) {
	chatID := message.ChatID(uuid.NewString())
	before := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	beforeID := message.ID(uuid.NewString())

	t.Run("applies defaults", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		repo.EXPECT().GetHistory(mock.Anything, mock.MatchedBy(func(q HistoryQuery) bool {
			return q.ChatID == chatID && q.Limit == 50 && !q.Before.IsZero()
		})).Return(nil, nil).Once()

		_, err := NewGetHistory(repo).Execute(context.Background(), HistoryQuery{ChatID: chatID})
		require.NoError(t, err)
	})

	t.Run("passes cursor", func(t *testing.T) {
		q := HistoryQuery{ChatID: chatID, Before: before, BeforeID: beforeID, Limit: 10}
		repo := NewMockMessageRepository(t)
		repo.EXPECT().GetHistory(mock.Anything, q).Return(nil, nil).Once()

		_, err := NewGetHistory(repo).Execute(context.Background(), q)
		require.NoError(t, err)
	})

	invalid := []struct {
		name    string
		query   HistoryQuery
		wantErr error
	}{
		{"invalid chat", HistoryQuery{ChatID: "bad"}, ErrInvalidChatID},
		{"cursor ID without time", HistoryQuery{ChatID: chatID, BeforeID: beforeID}, ErrInvalidHistoryCursor},
		{"invalid cursor ID", HistoryQuery{ChatID: chatID, Before: before, BeforeID: "bad"}, ErrInvalidHistoryCursor},
		{"negative limit", HistoryQuery{ChatID: chatID, Limit: -1}, ErrInvalidHistoryLimit},
		{"too large limit", HistoryQuery{ChatID: chatID, Limit: 101}, ErrInvalidHistoryLimit},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGetHistory(NewMockMessageRepository(t)).Execute(context.Background(), tt.query)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUploadFile_Execute(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)

	t.Run("uploads detected content", func(t *testing.T) {
		repo := NewMockFileRepository(t)
		repo.EXPECT().Upload(mock.Anything, "key", mock.Anything, "image/png").Return(nil).Once()

		err := NewUploadFileUseCase(repo).Execute(context.Background(), UploadFileInput{
			Key: "key", File: bytes.NewReader(png), Size: int64(len(png)), ContentType: "image/png",
		})
		require.NoError(t, err)
	})

	invalid := []struct {
		name    string
		input   UploadFileInput
		wantErr error
	}{
		{"empty key", UploadFileInput{File: strings.NewReader("x"), Size: 1, ContentType: "text/plain"}, ErrEmptyKey},
		{"empty file", UploadFileInput{Key: "k", ContentType: "text/plain"}, ErrEmptyFile},
		{"too large", UploadFileInput{Key: "k", File: strings.NewReader("x"), Size: MaxFileSize + 1, ContentType: "text/plain"}, ErrFileTooLarge},
		{"negative size", UploadFileInput{Key: "k", File: strings.NewReader("x"), Size: -1, ContentType: "text/plain"}, ErrInvalidFileSize},
		{"missing content type", UploadFileInput{Key: "k", File: strings.NewReader("x"), Size: 1}, ErrUnsupportedMIME},
		{"spoofed content type", UploadFileInput{Key: "k", File: bytes.NewReader(png), Size: int64(len(png)), ContentType: "application/pdf"}, ErrUnsupportedMIME},
		{"actually empty", UploadFileInput{Key: "k", File: strings.NewReader(""), Size: 10, ContentType: "text/plain"}, ErrEmptyFile},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUploadFileUseCase(NewMockFileRepository(t)).Execute(context.Background(), tt.input)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("storage fails", func(t *testing.T) {
		repo := NewMockFileRepository(t)
		repo.EXPECT().Upload(mock.Anything, "key", mock.Anything, "image/png").Return(errTest).Once()

		err := NewUploadFileUseCase(repo).Execute(context.Background(), UploadFileInput{
			Key: "key", File: bytes.NewReader(png), Size: int64(len(png)), ContentType: "image/png",
		})
		require.ErrorIs(t, err, ErrInternal)
	})
}

func TestDownloadFile_Execute(t *testing.T) {
	repo := NewMockFileRepository(t)
	repo.EXPECT().Download(mock.Anything, "key").Return(strings.NewReader("data"), nil).Once()
	repo.EXPECT().Download(mock.Anything, "missing").Return(nil, errTest).Once()

	uc := NewDownloadFileUseCase(repo)
	r, err := uc.Execute(context.Background(), "key")
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "data", string(data))

	_, err = uc.Execute(context.Background(), "missing")
	require.ErrorIs(t, err, ErrInternal)

	_, err = uc.Execute(context.Background(), " ")
	require.ErrorIs(t, err, ErrEmptyKey)
}
