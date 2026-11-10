package block

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		userID    user.ID
		blockedID user.ID
		createdAt time.Time
		wantErr   error
	}{
		{"valid", "user-1", "user-2", now, nil},
		{"empty user ID", "", "user-2", now, ErrUserIDEmpty},
		{"empty blocked ID", "user-1", "", now, ErrBlockedIDEmpty},
		{"cannot block self", "user-1", "user-1", now, ErrCannotBlockSelf},
		{"zero createdAt", "user-1", "user-2", time.Time{}, ErrCreatedAtZero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.userID, tt.blockedID, tt.createdAt)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, Block{}, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.userID, got.UserID())
			require.Equal(t, tt.blockedID, got.BlockedID())
			require.Equal(t, tt.createdAt, got.CreatedAt())
		})
	}
}
