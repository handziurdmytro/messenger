package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Save(ctx context.Context, u user.User) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO users (id, email, display_name, avatar_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO NOTHING`,
		string(u.ID),
		u.Email,
		u.DisplayName,
		u.AvatarURL,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id user.ID) (user.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, `
        SELECT id, email, display_name, avatar_url, created_at, updated_at
        FROM users WHERE id = $1`,
		string(id),
	))
}

func (r *UserRepository) FindByIDs(ctx context.Context, ids []user.ID) ([]user.User, error) {
	raw := make([]string, len(ids))
	for i, id := range ids {
		raw[i] = string(id)
	}

	rows, err := r.pool.Query(ctx, `
        SELECT id, email, display_name, avatar_url, created_at, updated_at
        FROM users WHERE id = ANY($1)`,
		raw,
	)
	if err != nil {
		return nil, fmt.Errorf("find by ids: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *UserRepository) Update(ctx context.Context, u user.User) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE users
        SET display_name = $1, avatar_url = $2, updated_at = $3
        WHERE id = $4`,
		u.DisplayName,
		u.AvatarURL,
		u.UpdatedAt,
		string(u.ID),
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]user.User, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, email, display_name, avatar_url, created_at, updated_at
        FROM users
        WHERE display_name ILIKE $1 OR email ILIKE $1
        LIMIT $2`,
		"%"+query+"%",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *UserRepository) scanUser(row pgx.Row) (user.User, error) {
	var u user.User
	var id, avatarURL string
	var updatedAt time.Time

	err := row.Scan(&id, &u.Email, &u.DisplayName, &avatarURL, &u.CreatedAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrUserNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("scan user: %w", err)
	}

	u.ID = user.ID(id)
	u.AvatarURL = avatarURL
	u.UpdatedAt = updatedAt
	return u, nil
}

func (r *UserRepository) scanUsers(rows pgx.Rows) ([]user.User, error) {
	var users []user.User
	for rows.Next() {
		var u user.User
		var id, avatarURL string
		var updatedAt time.Time

		if err := rows.Scan(&id, &u.Email, &u.DisplayName, &avatarURL, &u.CreatedAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}

		u.ID = user.ID(id)
		u.AvatarURL = avatarURL
		u.UpdatedAt = updatedAt
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) AddContact(ctx context.Context, userID, contactID user.ID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contacts (user_id, contact_id, created_at)
		VALUES ($1, $2, $3), ($2, $1, $3)
		ON CONFLICT (user_id, contact_id) DO NOTHING`,
		string(userID),
		string(contactID),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("add contact: %w", err)
	}
	return nil
}

func (r *UserRepository) GetContacts(ctx context.Context, userID user.ID) ([]user.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.avatar_url, u.created_at, u.updated_at
		FROM users u
		INNER JOIN contacts c ON u.id = c.contact_id
		WHERE c.user_id = $1
		ORDER BY u.display_name ASC`,
		string(userID),
	)
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}
