package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/pkg/pgkit/pgtest"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/stretchr/testify/require"
)

var testCreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newTestRepositories(t *testing.T) (*UserRepository, *ContactRepository) {
	t.Helper()

	pg := pgtest.NewPostgres(t)
	pgtest.Migrate(t, pg.DSN, "../../../../migrations")

	queries := db.New(pg.Pool)
	return NewUserRepository(queries), NewContactRepository(queries)
}

func newTestUser(t *testing.T, id user.ID, email string) user.User {
	t.Helper()

	u, err := user.New(id, email, testCreatedAt)
	require.NoError(t, err)
	return u
}

func saveTestUser(t *testing.T, ctx context.Context, users *UserRepository, id user.ID, email string) user.User {
	t.Helper()

	u := newTestUser(t, id, email)
	require.NoError(t, users.Save(ctx, u))
	return u
}

func requireSameUser(t *testing.T, expected, actual user.User) {
	t.Helper()

	require.Equal(t, expected.ID(), actual.ID())
	require.Equal(t, expected.Email().String(), actual.Email().String())
	require.Equal(t, expected.DisplayName(), actual.DisplayName())
	require.Equal(t, expected.AvatarURL(), actual.AvatarURL())
	require.True(t, expected.CreatedAt().Equal(actual.CreatedAt()))
	require.True(t, expected.UpdatedAt().Equal(actual.UpdatedAt()))
}

func TestUserRepository_Save(t *testing.T) {
	ctx := context.Background()
	users, _ := newTestRepositories(t)

	t.Run("creates user", func(t *testing.T) {
		u := newTestUser(t, "save-1", "save1@example.com")

		require.NoError(t, users.Save(ctx, u))

		found, err := users.FindByID(ctx, u.ID())
		require.NoError(t, err)
		requireSameUser(t, u, found)
	})

	t.Run("repeated save is idempotent", func(t *testing.T) {
		u := newTestUser(t, "save-2", "save2@example.com")
		require.NoError(t, users.Save(ctx, u))

		require.NoError(t, users.Save(ctx, u))

		found, err := users.FindByID(ctx, u.ID())
		require.NoError(t, err)
		requireSameUser(t, u, found)
	})

	t.Run("same id with different email", func(t *testing.T) {
		original := saveTestUser(t, ctx, users, "save-3", "save3@example.com")
		conflicting := newTestUser(t, original.ID(), "other3@example.com")

		require.ErrorIs(t, users.Save(ctx, conflicting), user.ErrAlreadyExists)

		found, err := users.FindByID(ctx, original.ID())
		require.NoError(t, err)
		requireSameUser(t, original, found)
	})

	t.Run("same id with different creation time", func(t *testing.T) {
		original := saveTestUser(t, ctx, users, "save-4", "save4@example.com")
		conflicting, err := user.New(original.ID(), original.Email().String(), testCreatedAt.Add(time.Hour))
		require.NoError(t, err)

		require.ErrorIs(t, users.Save(ctx, conflicting), user.ErrAlreadyExists)

		found, err := users.FindByID(ctx, original.ID())
		require.NoError(t, err)
		requireSameUser(t, original, found)
	})

	t.Run("email already belongs to another id", func(t *testing.T) {
		original := saveTestUser(t, ctx, users, "save-5", "save5@example.com")
		conflicting := newTestUser(t, "save-6", original.Email().String())

		require.ErrorIs(t, users.Save(ctx, conflicting), user.ErrEmailTaken)

		missing, err := users.FindByID(ctx, conflicting.ID())
		require.ErrorIs(t, err, user.ErrNotFound)
		require.Equal(t, user.User{}, missing)

		found, err := users.FindByID(ctx, original.ID())
		require.NoError(t, err)
		requireSameUser(t, original, found)
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	users, _ := newTestRepositories(t)

	t.Run("found", func(t *testing.T) {
		expected := saveTestUser(t, ctx, users, "find-1", "find1@example.com")

		found, err := users.FindByID(ctx, expected.ID())

		require.NoError(t, err)
		requireSameUser(t, expected, found)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := users.FindByID(ctx, "missing")

		require.ErrorIs(t, err, user.ErrNotFound)
		require.Equal(t, user.User{}, found)
	})
}

func TestUserRepository_FindByIDs(t *testing.T) {
	ctx := context.Background()
	users, _ := newTestRepositories(t)

	first := saveTestUser(t, ctx, users, "ids-1", "ids1@example.com")
	second := saveTestUser(t, ctx, users, "ids-2", "ids2@example.com")

	t.Run("multiple users", func(t *testing.T) {
		found, err := users.FindByIDs(ctx, []user.ID{first.ID(), second.ID()})

		require.NoError(t, err)
		require.Len(t, found, 2)
		byID := make(map[user.ID]user.User, len(found))
		for _, u := range found {
			byID[u.ID()] = u
		}
		requireSameUser(t, first, byID[first.ID()])
		requireSameUser(t, second, byID[second.ID()])
	})

	t.Run("ignores missing ids", func(t *testing.T) {
		found, err := users.FindByIDs(ctx, []user.ID{first.ID(), "missing"})

		require.NoError(t, err)
		require.Len(t, found, 1)
		requireSameUser(t, first, found[0])
	})

	t.Run("empty ids", func(t *testing.T) {
		found, err := users.FindByIDs(ctx, nil)

		require.NoError(t, err)
		require.Empty(t, found)
	})
}

func TestUserRepository_Update(t *testing.T) {
	ctx := context.Background()
	users, _ := newTestRepositories(t)

	t.Run("updates mutable fields", func(t *testing.T) {
		u := saveTestUser(t, ctx, users, "update-1", "update1@example.com")
		now := testCreatedAt.Add(time.Hour)
		require.NoError(t, u.ChangeDisplayName("Updated Name", now))
		require.NoError(t, u.ChangeAvatarURL("https://example.com/avatar.png", now))

		require.NoError(t, users.Update(ctx, u))

		found, err := users.FindByID(ctx, u.ID())
		require.NoError(t, err)
		requireSameUser(t, u, found)
	})

	t.Run("missing user", func(t *testing.T) {
		u := newTestUser(t, "update-missing", "update-missing@example.com")

		require.ErrorIs(t, users.Update(ctx, u), user.ErrNotFound)
	})
}

func TestUserRepository_Search(t *testing.T) {
	ctx := context.Background()
	users, _ := newTestRepositories(t)

	alice := saveTestUser(t, ctx, users, "search-1", "alice@example.com")
	bob := saveTestUser(t, ctx, users, "search-2", "bob@example.com")
	charlie := saveTestUser(t, ctx, users, "search-3", "charlie@example.com")
	require.NoError(t, charlie.ChangeDisplayName("Alice Cooper", testCreatedAt.Add(time.Minute)))
	require.NoError(t, users.Update(ctx, charlie))

	t.Run("email substring case insensitive", func(t *testing.T) {
		found, err := users.Search(ctx, "BOB@EXAMPLE", 10)

		require.NoError(t, err)
		require.Len(t, found, 1)
		requireSameUser(t, bob, found[0])
	})

	t.Run("display name substring case insensitive", func(t *testing.T) {
		found, err := users.Search(ctx, "COOPER", 10)

		require.NoError(t, err)
		require.Len(t, found, 1)
		requireSameUser(t, charlie, found[0])
	})

	t.Run("matches email and display name", func(t *testing.T) {
		found, err := users.Search(ctx, "alice", 10)

		require.NoError(t, err)
		require.Len(t, found, 2)
		ids := []user.ID{found[0].ID(), found[1].ID()}
		require.ElementsMatch(t, []user.ID{alice.ID(), charlie.ID()}, ids)
	})

	t.Run("respects limit", func(t *testing.T) {
		found, err := users.Search(ctx, "example.com", 2)

		require.NoError(t, err)
		require.Len(t, found, 2)
	})

	t.Run("no matches", func(t *testing.T) {
		found, err := users.Search(ctx, "no-such-user", 10)

		require.NoError(t, err)
		require.Empty(t, found)
	})
}
