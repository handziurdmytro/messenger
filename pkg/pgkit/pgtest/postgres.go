package pgtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/handziurdmytro/messenger/pkg/pgkit"
)

type Postgres struct {
	DSN  string
	Pool *pgxpool.Pool
}

func NewPostgres(t testing.TB) *Postgres {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	testcontainers.CleanupContainer(t, container)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	cfg := pgkit.DefaultConfig(dsn)
	cfg.MinIdleConns = 0

	pool, err := pgkit.NewPool(ctx, cfg)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return &Postgres{
		DSN:  dsn,
		Pool: pool,
	}
}
