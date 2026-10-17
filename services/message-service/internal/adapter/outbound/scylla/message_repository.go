package scylla

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var (
	ErrPersistence  = errors.New("scylla persistence error")
	ErrNotFound     = errors.New("message not found")
	ErrConflict     = errors.New("message ID conflict")
	ErrInvalidQuery = errors.New("invalid history query")
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 100
	earliestYear        = 1970
)

type MessageRepository struct {
	session *gocql.Session
}

func NewMessageRepository(hosts []string) (*MessageRepository, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = "messenger"
	cluster.Consistency = gocql.Quorum
	cluster.SerialConsistency = gocql.Serial
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, errkit.Wrap(ErrPersistence, "connect scylla", err)
	}
	return &MessageRepository{session: session}, nil
}

func (r *MessageRepository) Close() {
	r.session.Close()
}

func (r *MessageRepository) Save(ctx context.Context, msg message.Message) (message.Message, error) {
	id, err := gocql.ParseUUID(string(msg.ID))
	if err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "parse message ID", err)
	}
	if msg.CreatedAt.IsZero() || msg.EditedAt != nil || msg.DeletedAt != nil {
		return message.Message{}, ErrConflict
	}

	hash, err := contentHash(msg)
	if err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "hash message", err)
	}
	createdAt := msg.CreatedAt.UTC().Truncate(time.Millisecond)

	existing := map[string]any{}
	applied, err := r.session.Query(`
		INSERT INTO message_locations (chat_id, message_id, bucket, created_at, request_hash)
		VALUES (?, ?, ?, ?, ?) IF NOT EXISTS`,
		string(msg.ChatID), id, bucket(createdAt), createdAt, hash,
	).WithContext(ctx).MapScanCAS(existing)
	if err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "reserve message", err)
	}

	storedBucket, storedAt, storedHash := bucket(createdAt), createdAt, hash
	if !applied {
		storedBucket, storedAt, storedHash, err = locationFromRow(existing)
		if err != nil {
			return message.Message{}, err
		}
	}

	if storedHash != hash {
		return message.Message{}, ErrConflict
	}

	_, err = r.session.Query(`
		INSERT INTO messages_by_chat
		(chat_id, bucket, created_at, message_id, from_user, type, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?) IF NOT EXISTS`,
		string(msg.ChatID), storedBucket, storedAt, id,
		msg.FromUser, string(msg.Type), string(msg.Payload),
	).WithContext(ctx).MapScanCAS(map[string]any{})
	if err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "save message", err)
	}

	msg.CreatedAt = storedAt.UTC()
	return msg, nil
}

func (r *MessageRepository) FindByID(ctx context.Context, chatID message.ChatID, messageID message.ID) (message.Message, error) {
	id, err := gocql.ParseUUID(string(messageID))
	if err != nil {
		return message.Message{}, ErrNotFound
	}

	var b int
	var at time.Time
	err = r.session.Query(`
		SELECT bucket, created_at FROM message_locations
		WHERE chat_id = ? AND message_id = ?`,
		string(chatID), id,
	).WithContext(ctx).Scan(&b, &at)
	if err != nil {
		return message.Message{}, readError("find location", err)
	}

	var row messageRow
	err = r.session.Query(`
		SELECT message_id, from_user, type, payload, created_at, edited_at, deleted_at
		FROM messages_by_chat
		WHERE chat_id = ? AND bucket = ? AND created_at = ? AND message_id = ?`,
		string(chatID), b, at, id,
	).WithContext(ctx).Scan(
		&row.id, &row.sender, &row.typ, &row.payload,
		&row.createdAt, &row.editedAt, &row.deletedAt,
	)
	if err != nil {
		return message.Message{}, readError("find message", err)
	}
	return restore(chatID, row)
}

func (r *MessageRepository) GetHistory(ctx context.Context, q usecase.HistoryQuery) ([]message.Message, error) {
	if q.ChatID == "" || q.Limit < 0 || q.Limit > maxHistoryLimit || (q.Before.IsZero() && q.BeforeID != "") {
		return nil, ErrInvalidQuery
	}
	if q.Limit == 0 {
		q.Limit = defaultHistoryLimit
	}
	if q.Before.IsZero() {
		q.Before = time.Now().UTC()
	}

	var cursorID gocql.UUID
	if q.BeforeID != "" {
		var err error
		if cursorID, err = gocql.ParseUUID(string(q.BeforeID)); err != nil {
			return nil, ErrInvalidQuery
		}
	}

	cursor := q.Before.UTC()
	firstMonth := monthStart(cursor)
	result := make([]message.Message, 0, q.Limit)

	for month := firstMonth; month.Year() >= earliestYear && len(result) < q.Limit; month = month.AddDate(0, -1, 0) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		query := `
			SELECT message_id, from_user, type, payload, created_at, edited_at, deleted_at
			FROM messages_by_chat WHERE chat_id = ? AND bucket = ?`
		args := []any{string(q.ChatID), bucket(month)}

		if month.Equal(firstMonth) {
			if q.BeforeID != "" {
				query += " AND (created_at, message_id) < (?, ?)"
				args = append(args, cursor, cursorID)
			} else {
				query += " AND created_at < ?"
				args = append(args, cursor)
			}
		}

		remaining := q.Limit - len(result)
		query += " LIMIT ?"
		args = append(args, remaining)

		msgs, err := r.scanMessages(ctx, q.ChatID, remaining, query, args)
		if err != nil {
			return nil, err
		}
		result = append(result, msgs...)
	}
	return result, nil
}

func (r *MessageRepository) scanMessages(
	ctx context.Context, chatID message.ChatID, pageSize int, query string, args []any,
) ([]message.Message, error) {
	iter := r.session.Query(query, args...).WithContext(ctx).PageSize(pageSize).Iter()

	var msgs []message.Message
	for {
		var row messageRow
		if !iter.Scan(&row.id, &row.sender, &row.typ, &row.payload,
			&row.createdAt, &row.editedAt, &row.deletedAt) {
			break
		}
		msg, err := restore(chatID, row)
		if err != nil {
			_ = iter.Close()
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	if err := iter.Close(); err != nil {
		return nil, errkit.Wrap(ErrPersistence, "get history", err)
	}
	return msgs, nil
}

type messageRow struct {
	id        gocql.UUID
	sender    string
	typ       string
	payload   string
	createdAt time.Time
	editedAt  *time.Time
	deletedAt *time.Time
}

func restore(chatID message.ChatID, row messageRow) (message.Message, error) {
	var payload json.RawMessage
	if row.deletedAt == nil {
		payload = json.RawMessage(row.payload)
	}
	msg, err := message.Restore(
		message.ID(row.id.String()), chatID, row.sender, message.Type(row.typ),
		payload, row.createdAt, row.editedAt, row.deletedAt,
	)
	if err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "restore message", err)
	}
	return msg, nil
}

func contentHash(msg message.Message) (string, error) {
	data, err := json.Marshal(struct {
		ChatID   message.ChatID  `json:"chat_id"`
		FromUser string          `json:"from_user"`
		Type     message.Type    `json:"type"`
		Payload  json.RawMessage `json:"payload"`
	}{msg.ChatID, msg.FromUser, msg.Type, msg.Payload})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func locationFromRow(row map[string]any) (int, time.Time, string, error) {
	b, okB := row["bucket"].(int)
	at, okAt := row["created_at"].(time.Time)
	hash, okH := row["request_hash"].(string)
	if !okB || !okAt || !okH {
		return 0, time.Time{}, "", errkit.Wrap(ErrPersistence, "read location",
			fmt.Errorf("unexpected CAS row: %v", row))
	}
	return b, at.UTC(), hash, nil
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func bucket(t time.Time) int {
	t = t.UTC()
	return t.Year()*100 + int(t.Month())
}

func readError(operation string, err error) error {
	if errors.Is(err, gocql.ErrNotFound) {
		return ErrNotFound
	}
	return errkit.Wrap(ErrPersistence, operation, err)
}
