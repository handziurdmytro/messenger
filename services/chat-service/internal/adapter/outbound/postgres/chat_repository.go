package postgres

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
)

const constraintPrivateKey = "uq_chats_private_key"

type ChatRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *ChatRepository) Save(ctx context.Context, c chat.Chat) error {
	id, err := uuid.Parse(string(c.ID()))
	if err != nil {
		return errkit.Wrap(usecase.ErrPersistence, "postgres: parse chat ID", err)
	}

	if c.Type() != chat.TypePrivate {
		return chat.ErrUnsupportedChatType
	}

	participants := c.Participants()
	slices.Sort(participants)

	privateKey := pgtype.Text{
		String: fmt.Sprintf("%s:%s", participants[0], participants[1]),
		Valid:  true,
	}

	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		if err := q.CreateChat(ctx, db.CreateChatParams{
			ID:         id,
			Type:       string(c.Type()),
			PrivateKey: privateKey,
			CreatedAt:  c.CreatedAt(),
		}); err != nil {
			return err
		}

		for _, participant := range participants {
			userID, err := uuid.Parse(string(participant))
			if err != nil {
				return err
			}

			if err := q.AddParticipant(ctx, db.AddParticipantParams{
				ChatID: id,
				UserID: userID,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if pgkit.IsConstraint(err, pgkit.CodeUniqueViolation, constraintPrivateKey) {
			return errkit.Wrap(usecase.ErrChatAlreadyExists, "postgres: save chat", err)
		}
		return errkit.Wrap(usecase.ErrPersistence, "postgres: save chat", err)
	}

	return nil
}

func (r *ChatRepository) FindByID(ctx context.Context, id chat.ID) (chat.Chat, error) {
	parsedID, err := uuid.Parse(string(id))
	if err != nil {
		return chat.Chat{}, chat.ErrInvalidChatID
	}

	row, err := r.q.FindChatByID(ctx, parsedID)
	if err != nil {
		return chat.Chat{}, findError("postgres: find chat", err)
	}

	return r.restore(ctx, row.ID, row.Type, row.CreatedAt)
}

func (r *ChatRepository) FindPrivateByParticipants(ctx context.Context, u1, u2 chat.UserID) (chat.Chat, error) {
	participants := []chat.UserID{u1, u2}
	slices.Sort(participants)

	key := fmt.Sprintf("%s:%s", participants[0], participants[1])

	row, err := r.q.FindPrivateByParticipants(ctx, pgtype.Text{
		String: key,
		Valid:  true,
	})
	if err != nil {
		return chat.Chat{}, findError("postgres: find private chat", err)
	}

	return r.restore(ctx, row.ID, row.Type, row.CreatedAt)
}

func (r *ChatRepository) restore(ctx context.Context, id uuid.UUID, chatType string, createdAt time.Time) (chat.Chat, error) {
	userIDs, err := r.q.ListParticipantsByChatID(ctx, id)
	if err != nil {
		return chat.Chat{}, errkit.Wrap(usecase.ErrPersistence, "postgres: list participants", err)
	}

	participants := make([]chat.UserID, 0, len(userIDs))
	for _, userID := range userIDs {
		participants = append(participants, chat.UserID(userID.String()))
	}

	c, err := chat.Restore(chat.ID(id.String()), chat.Type(chatType), participants, createdAt)
	if err != nil {
		return chat.Chat{}, errkit.Wrap(usecase.ErrPersistence, "postgres: restore chat", err)
	}

	return c, nil
}

func findError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.ErrChatNotFound
	}
	return errkit.Wrap(usecase.ErrPersistence, operation, err)
}
