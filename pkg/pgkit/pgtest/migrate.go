package pgtest

import (
	"errors"
	"net/url"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func Migrate(t testing.TB, dsn, migrationsDir string) {
	t.Helper()

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   migrationsDir,
	}).String()

	databaseURL, err := url.Parse(dsn)
	require.NoError(t, err)

	databaseURL.Scheme = "pgx5"

	m, err := migrate.New(sourceURL, databaseURL.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = m.Close()
	})

	err = m.Up()
	require.True(t, err == nil || errors.Is(err, migrate.ErrNoChange), "run migrations: %v", err)
}
