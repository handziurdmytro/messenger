package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Save(ctx context.Context, user auth.User) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO users (id, email, password_hash, created_at)
        VALUES ($1, $2, $3, $4)`,
		string(user.ID),
		user.Email,
		string(user.PasswordHash),
		user.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return auth.ErrEmailTaken
		}
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	var id, emailVal, passwordHash string
	var user auth.User

	err := r.pool.QueryRow(ctx, `
        SELECT id, email, password_hash, created_at
        FROM users WHERE email = $1`,
		email,
	).Scan(&id, &emailVal, &passwordHash, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("find user by email: %w", err)
	}

	user.ID = auth.UserID(id)
	user.Email = emailVal
	user.PasswordHash = auth.PasswordHash(passwordHash)
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id auth.UserID) (auth.User, error) {
	var userID, email, passwordHash string
	var user auth.User

	err := r.pool.QueryRow(ctx, `
        SELECT id, email, password_hash, created_at
        FROM users WHERE id = $1`,
		string(id),
	).Scan(&userID, &email, &passwordHash, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("find user by id: %w", err)
	}

	user.ID = auth.UserID(userID)
	user.Email = email
	user.PasswordHash = auth.PasswordHash(passwordHash)
	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
