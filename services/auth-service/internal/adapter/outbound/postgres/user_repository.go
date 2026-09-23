package postgres

import (
	"context"
	"errors"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	"github.com/jackc/pgx/v5"
)

const usersEmailUniqueConstraint = "users_email_unique"

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Save(
	ctx context.Context,
	user auth.User,
) error {
	err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:           string(user.ID),
		Email:        user.Email,
		PasswordHash: string(user.PasswordHash),
		CreatedAt:    user.CreatedAt,
	})
	if err == nil {
		return nil
	}

	if pgkit.IsConstraint(
		err,
		pgkit.CodeUniqueViolation,
		usersEmailUniqueConstraint,
	) {
		return auth.ErrEmailTaken
	}

	return errkit.Wrap(outbound.ErrPersistence, "create user", err)
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (auth.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, errkit.Wrap(outbound.ErrPersistence, "find user by email", err)
	}

	return auth.User{
		ID:           auth.UserID(row.ID),
		Email:        row.Email,
		PasswordHash: auth.PasswordHash(row.PasswordHash),
		CreatedAt:    row.CreatedAt,
	}, nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id auth.UserID,
) (auth.User, error) {
	row, err := r.q.GetUserByID(ctx, string(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, errkit.Wrap(outbound.ErrPersistence, "find user by id", err)
	}

	return auth.User{
		ID:           auth.UserID(row.ID),
		Email:        row.Email,
		PasswordHash: auth.PasswordHash(row.PasswordHash),
		CreatedAt:    row.CreatedAt,
	}, nil
}
