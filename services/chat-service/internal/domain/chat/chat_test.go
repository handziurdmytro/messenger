package chat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUser1 UserID = "550e8400-e29b-41d4-a716-446655440001"
	testUser2 UserID = "550e8400-e29b-41d4-a716-446655440002"
)

func TestNewPrivate(t *testing.T) {
	before := time.Now().UTC()
	c, err := NewPrivate(testUser1, testUser2)
	after := time.Now().UTC()

	require.NoError(t, err)

	_, err = ParseID(string(c.ID()))
	require.NoError(t, err)

	assert.Equal(t, TypePrivate, c.Type())
	assert.Equal(t, []UserID{testUser1, testUser2}, c.Participants())
	assert.True(t, c.HasParticipant(testUser1))
	assert.True(t, c.HasParticipant(testUser2))
	assert.False(t, c.CreatedAt().Before(before))
	assert.False(t, c.CreatedAt().After(after))
}

func TestNewPrivateValidation(t *testing.T) {
	tests := []struct {
		name    string
		user1   UserID
		user2   UserID
		wantErr error
	}{
		{"same user", testUser1, testUser1, ErrInvalidParticipants},
		{"empty first", "", testUser2, ErrInvalidUserID},
		{"empty second", testUser1, "", ErrInvalidUserID},
		{"invalid first", "invalid", testUser2, ErrInvalidUserID},
		{"invalid second", testUser1, "invalid", ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPrivate(tt.user1, tt.user2)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRestore(t *testing.T) {
	id := ID("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	participants := []UserID{testUser1, testUser2}

	c, err := Restore(id, TypePrivate, participants, createdAt)
	require.NoError(t, err)

	assert.Equal(t, id, c.ID())
	assert.Equal(t, TypePrivate, c.Type())
	assert.Equal(t, createdAt, c.CreatedAt())
	assert.Equal(t, participants, c.Participants())

	participants[0] = testUser2
	assert.True(t, c.HasParticipant(testUser1), "input slice must be cloned")
}

func TestRestoreValidation(t *testing.T) {
	validID := ID("550e8400-e29b-41d4-a716-446655440000")
	now := time.Now().UTC()

	tests := []struct {
		name         string
		id           ID
		chatType     Type
		participants []UserID
		createdAt    time.Time
		wantErr      error
	}{
		{"invalid ID", "invalid", TypePrivate, []UserID{testUser1, testUser2}, now, ErrInvalidChatID},
		{"zero time", validID, TypePrivate, []UserID{testUser1, testUser2}, time.Time{}, ErrInvalidCreatedAt},
		{"unsupported type", validID, TypeGroup, []UserID{testUser1, testUser2}, now, ErrUnsupportedChatType},
		{"no participants", validID, TypePrivate, nil, now, ErrInvalidParticipants},
		{"one participant", validID, TypePrivate, []UserID{testUser1}, now, ErrInvalidParticipants},
		{"three participants", validID, TypePrivate, []UserID{testUser1, testUser2, testUser1}, now, ErrInvalidParticipants},
		{"duplicate participants", validID, TypePrivate, []UserID{testUser1, testUser1}, now, ErrInvalidParticipants},
		{"invalid participant", validID, TypePrivate, []UserID{testUser1, "invalid"}, now, ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Restore(tt.id, tt.chatType, tt.participants, tt.createdAt)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestParticipantsIsolation(t *testing.T) {
	c, err := NewPrivate(testUser1, testUser2)
	require.NoError(t, err)

	participants := c.Participants()
	participants[0] = testUser2

	assert.True(t, c.HasParticipant(testUser1))
	assert.Equal(t, []UserID{testUser1, testUser2}, c.Participants())
	assert.False(t, c.HasParticipant(UserID("550e8400-e29b-41d4-a716-446655440003")))
}
