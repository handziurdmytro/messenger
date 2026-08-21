package scylla

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
)

type MessageRepository struct {
	session *gocql.Session
}

func NewMessageRepository(hosts []string) (*MessageRepository, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("connect to scylla: %w", err)
	}

	if err := session.Query(createKeyspace).Exec(); err != nil {
		session.Close()
		return nil, fmt.Errorf("create keyspace: %w", err)
	}

	if err := session.Query(createMessagesTable).Exec(); err != nil {
		session.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	session.Close()

	cluster.Keyspace = "messenger"
	session, err = cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("connect to messenger keyspace: %w", err)
	}

	return &MessageRepository{session: session}, nil
}

func (r *MessageRepository) Save(ctx context.Context, msg message.Message) error {
	return r.session.Query(`
        INSERT INTO messages (room_id, created_at, message_id, from_user, type, body)
        VALUES (?, ?, ?, ?, ?, ?)`,
		string(msg.RoomID),
		msg.CreatedAt,
		string(msg.ID),
		msg.FromUser,
		msg.Type,
		msg.Body,
	).WithContext(ctx).Exec()
}

func (r *MessageRepository) GetHistory(ctx context.Context, q message.Query) ([]message.Message, error) {
	q = q.WithDefaults()

	iter := r.session.Query(`
        SELECT message_id, from_user, type, body, created_at
        FROM messages
        WHERE room_id = ? AND created_at < ?
        LIMIT ?`,
		string(q.RoomID),
		q.Before,
		q.Limit,
	).WithContext(ctx).Iter()

	var messages []message.Message
	var (
		id        string
		fromUser  string
		msgType   string
		body      string
		createdAt time.Time
	)

	for iter.Scan(&id, &fromUser, &msgType, &body, &createdAt) {
		messages = append(messages, message.Message{
			ID:        message.ID(id),
			RoomID:    q.RoomID,
			FromUser:  fromUser,
			Type:      msgType,
			Body:      body,
			CreatedAt: createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	return messages, nil
}

func (r *MessageRepository) Close() {
	r.session.Close()
}
