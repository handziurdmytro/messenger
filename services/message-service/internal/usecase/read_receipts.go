package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

const MaxUnreadCount = 100

type MarkRead struct {
	messages  MessageRepository
	markers   ReadMarkerRepository
	publisher MessagePublisher
}

func NewMarkRead(messages MessageRepository, markers ReadMarkerRepository, publisher MessagePublisher) *MarkRead {
	return &MarkRead{messages: messages, markers: markers, publisher: publisher}
}

func (uc *MarkRead) Execute(ctx context.Context, chatID message.ChatID, messageID message.ID, userID string) (ReadMarker, error) {
	if _, err := uuid.Parse(string(chatID)); err != nil {
		return ReadMarker{}, ErrInvalidChatID
	}
	if _, err := uuid.Parse(string(messageID)); err != nil {
		return ReadMarker{}, ErrInvalidMessageID
	}

	msg, err := uc.messages.FindByID(ctx, chatID, messageID)
	if err != nil {
		return ReadMarker{}, err
	}

	marker, err := uc.markers.MarkRead(ctx, chatID, ReadMarker{
		UserID:    userID,
		MessageID: msg.ID,
		ReadAt:    msg.CreatedAt,
	})
	if err != nil {
		return ReadMarker{}, errkit.Wrap(ErrPersistence, "mark read", err)
	}

	if marker.MessageID != msg.ID {
		return marker, nil
	}

	event := contract.ReadEvent{
		ChatID:    string(chatID),
		UserID:    userID,
		MessageID: string(marker.MessageID),
		ReadAt:    marker.ReadAt,
	}
	if err := uc.publisher.PublishRead(ctx, event); err != nil {
		return ReadMarker{}, errkit.Wrap(ErrPublishFailed, "publish message read", err)
	}

	return marker, nil
}

type ListReadMarkers struct {
	markers ReadMarkerRepository
}

func NewListReadMarkers(markers ReadMarkerRepository) *ListReadMarkers {
	return &ListReadMarkers{markers: markers}
}

func (uc *ListReadMarkers) Execute(ctx context.Context, chatID message.ChatID) ([]ReadMarker, error) {
	if _, err := uuid.Parse(string(chatID)); err != nil {
		return nil, ErrInvalidChatID
	}

	markers, err := uc.markers.ListReadMarkers(ctx, chatID)
	if err != nil {
		return nil, errkit.Wrap(ErrPersistence, "list read markers", err)
	}
	return markers, nil
}

type UnreadCount struct {
	Count  int
	Capped bool
}

type GetUnreadCount struct {
	markers ReadMarkerRepository
}

func NewGetUnreadCount(markers ReadMarkerRepository) *GetUnreadCount {
	return &GetUnreadCount{markers: markers}
}

func (uc *GetUnreadCount) Execute(ctx context.Context, chatID message.ChatID, userID string) (UnreadCount, error) {
	markers, err := (&ListReadMarkers{markers: uc.markers}).Execute(ctx, chatID)
	if err != nil {
		return UnreadCount{}, err
	}

	var mine ReadMarker
	for _, m := range markers {
		if m.UserID == userID {
			mine = m
			break
		}
	}

	count, err := uc.markers.CountUnread(ctx, chatID, userID, mine.ReadAt, MaxUnreadCount)
	if err != nil {
		return UnreadCount{}, errkit.Wrap(ErrPersistence, "count unread", err)
	}

	return UnreadCount{Count: count, Capped: count >= MaxUnreadCount}, nil
}
