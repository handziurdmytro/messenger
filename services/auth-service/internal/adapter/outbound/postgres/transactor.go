package postgres

import (
	"context"

	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Transactor struct {
	pool *pgxpool.Pool
}

func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

func (t *Transactor) WithinTransaction(ctx context.Context, fn func(outbound.Repositories) error) error {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "begin transaction", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := db.New(tx)

	repos := outbound.Repositories{
		Users:  NewUserRepository(q),
		Tokens: NewTokenRepository(q),
	}

	if err := fn(repos); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "commit transaction", err)
	}

	return nil
}
