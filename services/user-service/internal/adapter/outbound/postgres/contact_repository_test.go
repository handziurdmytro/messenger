package postgres

import (
	"context"
	"testing"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
	"github.com/stretchr/testify/require"
)

func newTestContact(t *testing.T, ownerID, targetID user.ID) contact.Contact {
	t.Helper()

	c, err := contact.New(ownerID, targetID, testCreatedAt)
	require.NoError(t, err)
	return c
}

func TestContactRepository_Add(t *testing.T) {
	ctx := context.Background()
	users, contacts := newTestRepositories(t)

	t.Run("creates contact in both directions", func(t *testing.T) {
		owner := saveTestUser(t, ctx, users, "add-1", "add1@example.com")
		target := saveTestUser(t, ctx, users, "add-2", "add2@example.com")
		c := newTestContact(t, owner.ID(), target.ID())

		require.NoError(t, contacts.Add(ctx, c))

		ownerContacts, err := contacts.GetByUserID(ctx, owner.ID())
		require.NoError(t, err)
		require.Len(t, ownerContacts, 1)
		requireSameUser(t, target, ownerContacts[0])

		targetContacts, err := contacts.GetByUserID(ctx, target.ID())
		require.NoError(t, err)
		require.Len(t, targetContacts, 1)
		requireSameUser(t, owner, targetContacts[0])
	})

	t.Run("duplicate contact in either direction", func(t *testing.T) {
		owner := saveTestUser(t, ctx, users, "duplicate-1", "duplicate1@example.com")
		target := saveTestUser(t, ctx, users, "duplicate-2", "duplicate2@example.com")
		require.NoError(t, contacts.Add(ctx, newTestContact(t, owner.ID(), target.ID())))

		require.ErrorIs(t,
			contacts.Add(ctx, newTestContact(t, owner.ID(), target.ID())),
			contact.ErrAlreadyExists,
		)
		require.ErrorIs(t,
			contacts.Add(ctx, newTestContact(t, target.ID(), owner.ID())),
			contact.ErrAlreadyExists,
		)

		ownerContacts, err := contacts.GetByUserID(ctx, owner.ID())
		require.NoError(t, err)
		require.Len(t, ownerContacts, 1)
		requireSameUser(t, target, ownerContacts[0])

		targetContacts, err := contacts.GetByUserID(ctx, target.ID())
		require.NoError(t, err)
		require.Len(t, targetContacts, 1)
		requireSameUser(t, owner, targetContacts[0])
	})

	t.Run("foreign key violation leaves no partial contact", func(t *testing.T) {
		owner := saveTestUser(t, ctx, users, "fk-1", "fk1@example.com")
		c := newTestContact(t, owner.ID(), "nonexistent")

		require.ErrorIs(t, contacts.Add(ctx, c), outbound.ErrPersistence)

		found, err := contacts.GetByUserID(ctx, owner.ID())
		require.NoError(t, err)
		require.Empty(t, found)
	})
}

func TestContactRepository_GetByUserID(t *testing.T) {
	ctx := context.Background()
	users, contacts := newTestRepositories(t)

	t.Run("empty list", func(t *testing.T) {
		owner := saveTestUser(t, ctx, users, "empty-1", "empty1@example.com")

		found, err := contacts.GetByUserID(ctx, owner.ID())

		require.NoError(t, err)
		require.Empty(t, found)
	})

	t.Run("returns contacts sorted by display name", func(t *testing.T) {
		owner := saveTestUser(t, ctx, users, "list-1", "list1@example.com")
		zoe := saveTestUser(t, ctx, users, "list-2", "zoe@example.com")
		amy := saveTestUser(t, ctx, users, "list-3", "amy@example.com")
		outsider := saveTestUser(t, ctx, users, "list-4", "outsider@example.com")

		require.NoError(t, contacts.Add(ctx, newTestContact(t, owner.ID(), zoe.ID())))
		require.NoError(t, contacts.Add(ctx, newTestContact(t, owner.ID(), amy.ID())))

		found, err := contacts.GetByUserID(ctx, owner.ID())
		require.NoError(t, err)
		require.Len(t, found, 2)
		requireSameUser(t, amy, found[0])
		requireSameUser(t, zoe, found[1])

		otherContacts, err := contacts.GetByUserID(ctx, outsider.ID())
		require.NoError(t, err)
		require.Empty(t, otherContacts)
	})

	t.Run("nonexistent owner has no contacts", func(t *testing.T) {
		found, err := contacts.GetByUserID(ctx, "missing-owner")

		require.NoError(t, err)
		require.Empty(t, found)
	})
}
