package chat

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	assert.NotEqual(t, id1, id2)
	_, err := uuid.Parse(string(id1))
	require.NoError(t, err)
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid", "550e8400-e29b-41d4-a716-446655440000", nil},
		{"empty", "", ErrInvalidChatID},
		{"invalid", "not-a-uuid", ErrInvalidChatID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseID(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, id)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, ID(tt.input), id)
		})
	}
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid", "550e8400-e29b-41d4-a716-446655440001", nil},
		{"empty", "", ErrInvalidUserID},
		{"invalid", "user-123", ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseUserID(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, id)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, UserID(tt.input), id)
		})
	}
}
