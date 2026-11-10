package scylla

import (
	"context"
	"time"

	"github.com/gocql/gocql"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

const maxUnreadScanMonths = 24

func (r *MessageRepository) MarkRead(ctx context.Context, chatID message.ChatID, marker usecase.ReadMarker) (usecase.ReadMarker, error) {
	id, err := gocql.ParseUUID(string(marker.MessageID))
	if err != nil {
		return usecase.ReadMarker{}, ErrNotFound
	}
	readAt := marker.ReadAt.UTC().Truncate(time.Millisecond)

	existing := map[string]any{}
	applied, err := r.session.Query(`
		INSERT INTO read_markers (chat_id, user_id, message_id, read_at)
		VALUES (?, ?, ?, ?) IF NOT EXISTS`,
		string(chatID), marker.UserID, id, readAt,
	).WithContext(ctx).MapScanCAS(existing)
	if err != nil {
		return usecase.ReadMarker{}, errkit.Wrap(ErrPersistence, "insert read marker", err)
	}
	if applied {
		return usecase.ReadMarker{UserID: marker.UserID, MessageID: marker.MessageID, ReadAt: readAt}, nil
	}

	applied, err = r.session.Query(`
		UPDATE read_markers SET message_id = ?, read_at = ?
		WHERE chat_id = ? AND user_id = ?
		IF read_at < ?`,
		id, readAt, string(chatID), marker.UserID, readAt,
	).WithContext(ctx).MapScanCAS(map[string]any{})
	if err != nil {
		return usecase.ReadMarker{}, errkit.Wrap(ErrPersistence, "advance read marker", err)
	}
	if applied {
		return usecase.ReadMarker{UserID: marker.UserID, MessageID: marker.MessageID, ReadAt: readAt}, nil
	}

	var (
		storedID gocql.UUID
		storedAt time.Time
	)
	err = r.session.Query(`
		SELECT message_id, read_at FROM read_markers
		WHERE chat_id = ? AND user_id = ?`,
		string(chatID), marker.UserID,
	).WithContext(ctx).Scan(&storedID, &storedAt)
	if err != nil {
		return usecase.ReadMarker{}, errkit.Wrap(ErrPersistence, "read current marker", err)
	}

	return usecase.ReadMarker{
		UserID:    marker.UserID,
		MessageID: message.ID(storedID.String()),
		ReadAt:    storedAt.UTC(),
	}, nil
}

func (r *MessageRepository) ListReadMarkers(ctx context.Context, chatID message.ChatID) ([]usecase.ReadMarker, error) {
	iter := r.session.Query(`
		SELECT user_id, message_id, read_at FROM read_markers WHERE chat_id = ?`,
		string(chatID),
	).WithContext(ctx).Iter()

	var (
		markers []usecase.ReadMarker
		userID  string
		id      gocql.UUID
		readAt  time.Time
	)
	for iter.Scan(&userID, &id, &readAt) {
		markers = append(markers, usecase.ReadMarker{
			UserID:    userID,
			MessageID: message.ID(id.String()),
			ReadAt:    readAt.UTC(),
		})
	}
	if err := iter.Close(); err != nil {
		return nil, errkit.Wrap(ErrPersistence, "list read markers", err)
	}
	return markers, nil
}

func (r *MessageRepository) CountUnread(
	ctx context.Context, chatID message.ChatID, userID string, since time.Time, limit int,
) (int, error) {
	now := time.Now().UTC()
	oldest := monthStart(now).AddDate(0, -maxUnreadScanMonths+1, 0)
	if since.After(oldest) {
		oldest = monthStart(since)
	}

	count := 0
	for month := monthStart(now); !month.Before(oldest) && count < limit; month = month.AddDate(0, -1, 0) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		iter := r.session.Query(`
			SELECT from_user, deleted_at FROM messages_by_chat
			WHERE chat_id = ? AND bucket = ? AND created_at > ?`,
			string(chatID), bucket(month), since.UTC(),
		).WithContext(ctx).PageSize(limit).Iter()

		var (
			sender    string
			deletedAt *time.Time
		)
		for count < limit && iter.Scan(&sender, &deletedAt) {
			if sender != userID && deletedAt == nil {
				count++
			}
		}
		if err := iter.Close(); err != nil {
			return 0, errkit.Wrap(ErrPersistence, "count unread", err)
		}
	}
	return count, nil
}
