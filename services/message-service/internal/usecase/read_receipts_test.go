package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
)

func TestMarkRead_Execute(t *testing.T) {
	reader := uuid.NewString()
	msg := newTestMessage(t, uuid.NewString(), message.TypeText, `{"text":"hi"}`)
	requested := ReadMarker{UserID: reader, MessageID: msg.ID, ReadAt: msg.CreatedAt}

	t.Run("advances marker and publishes", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		markers := NewMockReadMarkerRepository(t)
		pub := NewMockMessagePublisher(t)
		repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()
		markers.EXPECT().MarkRead(mock.Anything, msg.ChatID, requested).Return(requested, nil).Once()
		pub.EXPECT().PublishRead(mock.Anything, contract.ReadEvent{
			ChatID: string(msg.ChatID), UserID: reader, MessageID: string(msg.ID), ReadAt: msg.CreatedAt,
		}).Return(nil).Once()

		got, err := NewMarkRead(repo, markers, pub).Execute(context.Background(), msg.ChatID, msg.ID, reader)
		require.NoError(t, err)
		require.Equal(t, requested, got)
	})

	t.Run("older message keeps marker without event", func(t *testing.T) {
		newer := ReadMarker{UserID: reader, MessageID: message.ID(uuid.NewString()), ReadAt: msg.CreatedAt.Add(time.Minute)}
		repo := NewMockMessageRepository(t)
		markers := NewMockReadMarkerRepository(t)
		repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()
		markers.EXPECT().MarkRead(mock.Anything, msg.ChatID, requested).Return(newer, nil).Once()

		got, err := NewMarkRead(repo, markers, NewMockMessagePublisher(t)).Execute(context.Background(), msg.ChatID, msg.ID, reader)
		require.NoError(t, err)
		require.Equal(t, newer, got)
	})

	t.Run("message not found", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(message.Message{}, ErrMessageNotFound).Once()

		_, err := NewMarkRead(repo, NewMockReadMarkerRepository(t), NewMockMessagePublisher(t)).Execute(context.Background(), msg.ChatID, msg.ID, reader)
		require.ErrorIs(t, err, ErrMessageNotFound)
	})

	t.Run("marker repository fails", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		markers := NewMockReadMarkerRepository(t)
		repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()
		markers.EXPECT().MarkRead(mock.Anything, msg.ChatID, requested).Return(ReadMarker{}, errTest).Once()

		_, err := NewMarkRead(repo, markers, NewMockMessagePublisher(t)).Execute(context.Background(), msg.ChatID, msg.ID, reader)
		require.ErrorIs(t, err, ErrPersistence)
	})

	t.Run("publish fails", func(t *testing.T) {
		repo := NewMockMessageRepository(t)
		markers := NewMockReadMarkerRepository(t)
		pub := NewMockMessagePublisher(t)
		repo.EXPECT().FindByID(mock.Anything, msg.ChatID, msg.ID).Return(msg, nil).Once()
		markers.EXPECT().MarkRead(mock.Anything, msg.ChatID, requested).Return(requested, nil).Once()
		pub.EXPECT().PublishRead(mock.Anything, mock.Anything).Return(errTest).Once()

		_, err := NewMarkRead(repo, markers, pub).Execute(context.Background(), msg.ChatID, msg.ID, reader)
		require.ErrorIs(t, err, ErrPublishFailed)
	})

	t.Run("invalid IDs", func(t *testing.T) {
		uc := NewMarkRead(NewMockMessageRepository(t), NewMockReadMarkerRepository(t), NewMockMessagePublisher(t))

		_, err := uc.Execute(context.Background(), "bad", msg.ID, reader)
		require.ErrorIs(t, err, ErrInvalidChatID)
		_, err = uc.Execute(context.Background(), msg.ChatID, "bad", reader)
		require.ErrorIs(t, err, ErrInvalidMessageID)
	})
}

func TestGetUnreadCount_Execute(t *testing.T) {
	chatID := message.ChatID(uuid.NewString())
	me := uuid.NewString()
	readAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	markers := []ReadMarker{
		{UserID: uuid.NewString(), ReadAt: readAt.Add(time.Hour)},
		{UserID: me, ReadAt: readAt},
	}

	tests := []struct {
		name    string
		markers []ReadMarker
		since   time.Time
		count   int
		want    UnreadCount
	}{
		{"counts since own marker", markers, readAt, 3, UnreadCount{Count: 3}},
		{"counts everything without marker", nil, time.Time{}, 7, UnreadCount{Count: 7}},
		{"caps the count", markers, readAt, MaxUnreadCount, UnreadCount{Count: MaxUnreadCount, Capped: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockReadMarkerRepository(t)
			repo.EXPECT().ListReadMarkers(mock.Anything, chatID).Return(tt.markers, nil).Once()
			repo.EXPECT().CountUnread(mock.Anything, chatID, me, tt.since, MaxUnreadCount).Return(tt.count, nil).Once()

			got, err := NewGetUnreadCount(repo).Execute(context.Background(), chatID, me)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	t.Run("count fails", func(t *testing.T) {
		repo := NewMockReadMarkerRepository(t)
		repo.EXPECT().ListReadMarkers(mock.Anything, chatID).Return(nil, nil).Once()
		repo.EXPECT().CountUnread(mock.Anything, chatID, me, time.Time{}, MaxUnreadCount).Return(0, errTest).Once()

		_, err := NewGetUnreadCount(repo).Execute(context.Background(), chatID, me)
		require.ErrorIs(t, err, ErrPersistence)
	})

	t.Run("invalid chat", func(t *testing.T) {
		_, err := NewGetUnreadCount(NewMockReadMarkerRepository(t)).Execute(context.Background(), "bad", me)
		require.ErrorIs(t, err, ErrInvalidChatID)
	})
}

func TestListReadMarkers_Execute(t *testing.T) {
	chatID := message.ChatID(uuid.NewString())
	markers := []ReadMarker{{UserID: uuid.NewString(), MessageID: message.ID(uuid.NewString())}}

	repo := NewMockReadMarkerRepository(t)
	repo.EXPECT().ListReadMarkers(mock.Anything, chatID).Return(markers, nil).Once()
	repo.EXPECT().ListReadMarkers(mock.Anything, chatID).Return(nil, errTest).Once()

	uc := NewListReadMarkers(repo)
	got, err := uc.Execute(context.Background(), chatID)
	require.NoError(t, err)
	require.Equal(t, markers, got)

	_, err = uc.Execute(context.Background(), chatID)
	require.ErrorIs(t, err, ErrPersistence)
}
