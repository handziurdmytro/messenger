package contact

import (
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		userID    user.ID
		contactID user.ID
		createdAt time.Time
		wantErr   error
	}{
		{"valid", "user-1", "user-2", now, nil},
		{"empty user ID", "", "user-2", now, ErrUserIDEmpty},
		{"empty contact ID", "user-1", "", now, ErrContactIDEmpty},
		{"cannot add self", "user-1", "user-1", now, ErrCannotAddSelf},
		{"zero createdAt", "user-1", "user-2", time.Time{}, ErrCreatedAtZero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.userID, tt.contactID, tt.createdAt)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, Contact{}, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.userID, got.UserID())
			require.Equal(t, tt.contactID, got.ContactID())
			require.Equal(t, tt.createdAt, got.CreatedAt())
		})
	}
}

func TestRestore(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	got, err := Restore("user-1", "user-2", now)

	require.NoError(t, err)
	require.Equal(t, user.ID("user-1"), got.UserID())
	require.Equal(t, user.ID("user-2"), got.ContactID())
	require.Equal(t, now, got.CreatedAt())
}
