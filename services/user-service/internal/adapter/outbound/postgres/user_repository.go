package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
	"github.com/jackc/pgx/v5"
)

const usersEmailUniqueConstraint = "users_email_unique"

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Save(ctx context.Context, u user.User) error {
	rowsAffected, err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:          u.ID().String(),
		Email:       u.Email().String(),
		DisplayName: u.DisplayName(),
		AvatarUrl:   u.AvatarURL(),
		CreatedAt:   u.CreatedAt(),
		UpdatedAt:   u.UpdatedAt(),
	})
	if err != nil {
		if pgkit.IsConstraint(err, pgkit.CodeUniqueViolation, usersEmailUniqueConstraint) {
			return user.ErrEmailTaken
		}
		return errkit.Wrap(outbound.ErrPersistence, "save user", err)
	}

	if rowsAffected == 0 {
		return user.ErrAlreadyExists
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id user.ID) (user.User, error) {
	row, err := r.q.GetUserByID(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, errkit.Wrap(outbound.ErrPersistence, "find user by id", err)
	}

	u, err := toDomainUser(row)
	if err != nil {
		return user.User{}, fmt.Errorf("restore user %s: %w", id, err)
	}
	return u, nil
}

func (r *UserRepository) FindByIDs(ctx context.Context, ids []user.ID) ([]user.User, error) {
	raw := make([]string, len(ids))
	for i, id := range ids {
		raw[i] = id.String()
	}

	rows, err := r.q.GetUsersByIDs(ctx, raw)
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "find users by ids", err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		u, err := toDomainUser(row)
		if err != nil {
			return nil, fmt.Errorf("restore user %s: %w", row.ID, err)
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, u user.User) error {
	rowsAffected, err := r.q.UpdateUser(ctx, db.UpdateUserParams{
		DisplayName: u.DisplayName(), AvatarUrl: u.AvatarURL(), UpdatedAt: u.UpdatedAt(), ID: u.ID().String(),
	})

	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "update user", err)
	}

	if rowsAffected == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]user.User, error) {
	rows, err := r.q.SearchUsers(ctx, db.SearchUsersParams{Query: query, LimitCount: int32(limit)})
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "search users", err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		u, err := toDomainUser(row)
		if err != nil {
			return nil, fmt.Errorf("restore user %s: %w", row.ID, err)
		}
		users = append(users, u)
	}

	return users, nil
}

func toDomainUser(row db.User) (user.User, error) {
	return user.Restore(user.ID(row.ID), row.Email, row.DisplayName, row.AvatarUrl, row.CreatedAt, row.UpdatedAt)
}
