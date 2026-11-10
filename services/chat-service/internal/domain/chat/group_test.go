package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUser3 UserID = "550e8400-e29b-41d4-a716-446655440003"

func newTestGroup(t *testing.T) Chat {
	t.Helper()

	c, err := NewGroup(testUser1, "  Team  ", []UserID{testUser2, testUser3, testUser2, testUser1})
	require.NoError(t, err)
	return c
}

func TestNewGroup(t *testing.T) {
	c := newTestGroup(t)

	assert.Equal(t, TypeGroup, c.Type())
	assert.Equal(t, "Team", c.Title())
	assert.Equal(t, testUser1, c.Owner())
	assert.Equal(t, []UserID{testUser1, testUser2, testUser3}, c.Participants())
}

func TestNewGroupValidation(t *testing.T) {
	tooMany := make([]UserID, 0, MaxGroupParticipants)
	for range MaxGroupParticipants {
		tooMany = append(tooMany, UserID(NewID()))
	}

	tests := []struct {
		name    string
		owner   UserID
		title   string
		members []UserID
		wantErr error
	}{
		{"invalid owner", "invalid", "Team", []UserID{testUser2}, ErrInvalidUserID},
		{"empty title", testUser1, "  ", []UserID{testUser2}, ErrInvalidTitle},
		{"long title", testUser1, strings.Repeat("a", MaxTitleLength+1), []UserID{testUser2}, ErrInvalidTitle},
		{"no members", testUser1, "Team", nil, ErrInvalidParticipants},
		{"only owner", testUser1, "Team", []UserID{testUser1}, ErrInvalidParticipants},
		{"invalid member", testUser1, "Team", []UserID{"invalid"}, ErrInvalidUserID},
		{"too many", testUser1, "Team", tooMany, ErrTooManyParticipants},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGroup(tt.owner, tt.title, tt.members)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRestoreGroup(t *testing.T) {
	id := ID("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	c, err := RestoreGroup(id, "Team", testUser2, []UserID{testUser1, testUser2}, createdAt)
	require.NoError(t, err)
	assert.Equal(t, testUser2, c.Owner())

	_, err = RestoreGroup(id, "Team", testUser3, []UserID{testUser1, testUser2}, createdAt)
	require.ErrorIs(t, err, ErrInvalidParticipants)

	_, err = RestoreGroup(id, "Team", testUser1, []UserID{testUser1, testUser1}, createdAt)
	require.ErrorIs(t, err, ErrInvalidParticipants)
}

func TestGroupManagement(t *testing.T) {
	c := newTestGroup(t)
	outsider := UserID("550e8400-e29b-41d4-a716-446655440004")

	require.ErrorIs(t, c.Rename(testUser2, "New"), ErrNotOwner)
	require.ErrorIs(t, c.Rename(outsider, "New"), ErrNotParticipant)
	require.NoError(t, c.Rename(testUser1, " New "))
	assert.Equal(t, "New", c.Title())

	require.ErrorIs(t, c.AddParticipants(testUser1, []UserID{testUser2}), ErrAlreadyParticipant)
	require.NoError(t, c.AddParticipants(testUser1, []UserID{outsider, testUser2}))
	assert.True(t, c.HasParticipant(outsider))

	require.ErrorIs(t, c.RemoveParticipant(testUser2, outsider), ErrNotOwner)
	require.ErrorIs(t, c.RemoveParticipant(testUser1, testUser1), ErrCannotRemoveOwner)
	require.NoError(t, c.RemoveParticipant(testUser1, outsider))
	require.ErrorIs(t, c.RemoveParticipant(testUser1, outsider), ErrNotParticipant)
}

func TestGroupLeave(t *testing.T) {
	c := newTestGroup(t)

	require.NoError(t, c.Leave(testUser1))
	assert.Equal(t, testUser2, c.Owner())
	assert.Equal(t, []UserID{testUser2, testUser3}, c.Participants())

	require.NoError(t, c.Leave(testUser3))
	require.NoError(t, c.Leave(testUser2))
	assert.True(t, c.IsEmpty())

	require.ErrorIs(t, c.Leave(testUser2), ErrNotParticipant)

	private, err := NewPrivate(testUser1, testUser2)
	require.NoError(t, err)
	require.ErrorIs(t, private.Leave(testUser1), ErrNotGroup)
	require.ErrorIs(t, private.Rename(testUser1, "x"), ErrNotGroup)
}
