package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"

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

const (
	roleOwner  = "owner"
	roleMember = "member"
)

type chatRecord struct {
	id           uuid.UUID
	chatType     string
	title        pgtype.Text
	createdAt    time.Time
	participants []uuid.UUID
	owner        uuid.UUID
}

func (r *ChatRepository) Save(ctx context.Context, c chat.Chat) error {
	id, err := uuid.Parse(string(c.ID()))
	if err != nil {
		return errkit.Wrap(usecase.ErrPersistence, "postgres: parse chat ID", err)
	}

	params := db.CreateChatParams{
		ID:        id,
		Type:      string(c.Type()),
		CreatedAt: c.CreatedAt(),
	}

	participants := c.Participants()
	switch c.Type() {
	case chat.TypePrivate:
		slices.Sort(participants)
		params.PrivateKey = pgtype.Text{
			String: fmt.Sprintf("%s:%s", participants[0], participants[1]),
			Valid:  true,
		}
	case chat.TypeGroup:
		params.Title = pgtype.Text{String: c.Title(), Valid: true}
	default:
		return chat.ErrUnsupportedChatType
	}

	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		if err := q.CreateChat(ctx, params); err != nil {
			return err
		}

		for _, participant := range participants {
			if err := addParticipant(ctx, q, id, participant, roleFor(c, participant)); err != nil {
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

func (r *ChatRepository) Update(ctx context.Context, c chat.Chat) error {
	id, err := uuid.Parse(string(c.ID()))
	if err != nil {
		return chat.ErrInvalidChatID
	}

	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		if err := q.UpdateChatTitle(ctx, db.UpdateChatTitleParams{
			ID:    id,
			Title: pgtype.Text{String: c.Title(), Valid: c.Title() != ""},
		}); err != nil {
			return err
		}

		current, err := q.ListParticipantsByChatID(ctx, id)
		if err != nil {
			return err
		}

		existing := make(map[chat.UserID]bool, len(current))
		for _, row := range current {
			userID := chat.UserID(row.UserID.String())
			existing[userID] = true
			if !c.HasParticipant(userID) {
				if err := q.RemoveParticipant(ctx, db.RemoveParticipantParams{ChatID: id, UserID: row.UserID}); err != nil {
					return err
				}
			}
		}

		for _, participant := range c.Participants() {
			role := roleFor(c, participant)
			if !existing[participant] {
				if err := addParticipant(ctx, q, id, participant, role); err != nil {
					return err
				}
				continue
			}

			userID, err := uuid.Parse(string(participant))
			if err != nil {
				return err
			}
			if err := q.SetParticipantRole(ctx, db.SetParticipantRoleParams{ChatID: id, UserID: userID, Role: role}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return errkit.Wrap(usecase.ErrPersistence, "postgres: update chat", err)
	}

	return nil
}

func addParticipant(ctx context.Context, q *db.Queries, chatID uuid.UUID, participant chat.UserID, role string) error {
	userID, err := uuid.Parse(string(participant))
	if err != nil {
		return err
	}

	return q.AddParticipant(ctx, db.AddParticipantParams{
		ChatID: chatID,
		UserID: userID,
		Role:   role,
	})
}

func roleFor(c chat.Chat, participant chat.UserID) string {
	if c.Type() == chat.TypeGroup && participant == c.Owner() {
		return roleOwner
	}
	return roleMember
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

	return r.restore(ctx, chatRecord{id: row.ID, chatType: row.Type, title: row.Title, createdAt: row.CreatedAt})
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

	return r.restore(ctx, chatRecord{id: row.ID, chatType: row.Type, title: row.Title, createdAt: row.CreatedAt})
}

func (r *ChatRepository) restore(ctx context.Context, record chatRecord) (chat.Chat, error) {
	rows, err := r.q.ListParticipantsByChatID(ctx, record.id)
	if err != nil {
		return chat.Chat{}, errkit.Wrap(usecase.ErrPersistence, "postgres: list participants", err)
	}

	for _, row := range rows {
		record.participants = append(record.participants, row.UserID)
		if row.Role == roleOwner {
			record.owner = row.UserID
		}
	}

	return toDomain(record)
}

func toDomain(record chatRecord) (chat.Chat, error) {
	participants := make([]chat.UserID, 0, len(record.participants))
	for _, userID := range record.participants {
		participants = append(participants, chat.UserID(userID.String()))
	}

	id := chat.ID(record.id.String())

	var (
		c   chat.Chat
		err error
	)
	if chat.Type(record.chatType) == chat.TypeGroup {
		c, err = chat.RestoreGroup(id, record.title.String, chat.UserID(record.owner.String()), participants, record.createdAt)
	} else {
		c, err = chat.Restore(id, chat.Type(record.chatType), participants, record.createdAt)
	}
	if err != nil {
		return chat.Chat{}, errkit.Wrap(usecase.ErrPersistence, "postgres: restore chat", err)
	}

	return c, nil
}

func (r *ChatRepository) Delete(ctx context.Context, id chat.ID) error {
	parsedID, err := uuid.Parse(string(id))
	if err != nil {
		return chat.ErrInvalidChatID
	}

	rows, err := r.q.DeleteChat(ctx, parsedID)
	if err != nil {
		return errkit.Wrap(usecase.ErrPersistence, "postgres: delete chat", err)
	}
	if rows == 0 {
		return usecase.ErrChatNotFound
	}

	return nil
}

func findError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.ErrChatNotFound
	}
	return errkit.Wrap(usecase.ErrPersistence, operation, err)
}

func (r *ChatRepository) ListByUser(ctx context.Context, userID chat.UserID, page usecase.ChatPage) ([]chat.Chat, error) {
	parsedID, err := uuid.Parse(string(userID))
	if err != nil {
		return nil, errkit.Wrap(chat.ErrInvalidUserID, "postgres: parse user ID", err)
	}

	if page.Limit <= 0 || page.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("postgres: invalid limit %d", page.Limit)
	}

	params := db.ListChatsByUserParams{
		UserID:     parsedID,
		LimitCount: int32(page.Limit),
	}
	if page.Before != nil {
		beforeID, err := uuid.Parse(string(page.Before.ID))
		if err != nil {
			return nil, errkit.Wrap(chat.ErrInvalidChatID, "postgres: parse cursor ID", err)
		}
		params.BeforeCreatedAt = pgtype.Timestamptz{Time: page.Before.CreatedAt, Valid: true}
		params.BeforeID = pgtype.UUID{Bytes: beforeID, Valid: true}
	}

	rows, err := r.q.ListChatsByUser(ctx, params)
	if err != nil {
		return nil, errkit.Wrap(usecase.ErrPersistence, "postgres: list chats", err)
	}

	chats := make([]chat.Chat, 0, len(rows))
	for _, row := range rows {
		record := chatRecord{
			id:           row.ID,
			chatType:     row.Type,
			title:        row.Title,
			createdAt:    row.CreatedAt,
			participants: row.ParticipantIds,
		}
		if len(row.OwnerIds) > 0 {
			record.owner = row.OwnerIds[0]
		}

		c, err := toDomain(record)
		if err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}

	return chats, nil
}
