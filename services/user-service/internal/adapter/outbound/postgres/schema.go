package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const createSchema = `
    CREATE TABLE IF NOT EXISTS users (
        id           TEXT PRIMARY KEY,
        email        TEXT UNIQUE NOT NULL,
        display_name TEXT NOT NULL,
        avatar_url   TEXT NOT NULL DEFAULT '',
        created_at   TIMESTAMPTZ NOT NULL,
        updated_at   TIMESTAMPTZ NOT NULL
    );

    CREATE TABLE IF NOT EXISTS contacts (
        user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        contact_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        created_at TIMESTAMPTZ NOT NULL,
        PRIMARY KEY (user_id, contact_id)
    );

    CREATE INDEX IF NOT EXISTS idx_users_display_name ON users USING gin(display_name gin_trgm_ops);
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm`); err != nil {
		return fmt.Errorf("create extension: %w", err)
	}

	if _, err := pool.Exec(ctx, createSchema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}
