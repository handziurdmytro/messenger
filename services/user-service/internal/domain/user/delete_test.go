package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUser_Delete(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := createdAt.Add(time.Hour)

	u, err := New("user-1", "alice@example.com", createdAt)
	require.NoError(t, err)
	require.NoError(t, u.ChangeAvatarURL("https://example.com/a.png", createdAt.Add(time.Minute)))

	require.ErrorIs(t, u.Delete(createdAt.Add(-time.Hour)), ErrInvalidTimestamp)
	require.ErrorIs(t, u.Delete(time.Time{}), ErrInvalidTimestamp)

	require.NoError(t, u.Delete(deletedAt))
	require.True(t, u.IsDeleted())
	require.Equal(t, deletedAt, u.DeletedAt())
	require.Equal(t, deletedAt, u.UpdatedAt())
	require.Equal(t, DeletedDisplayName, u.DisplayName())
	require.Empty(t, u.Email().String())
	require.Empty(t, u.AvatarURL())

	require.ErrorIs(t, u.Delete(deletedAt.Add(time.Hour)), ErrDeleted)
	require.ErrorIs(t, u.ChangeDisplayName("Bob", deletedAt.Add(time.Hour)), ErrDeleted)
	require.ErrorIs(t, u.ChangeAvatarURL("", deletedAt.Add(time.Hour)), ErrDeleted)
}

func TestRestoreDeleted(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := createdAt.Add(time.Hour)

	tests := []struct {
		name      string
		id        ID
		createdAt time.Time
		updatedAt time.Time
		deletedAt time.Time
		wantErr   error
	}{
		{"valid", "user-1", createdAt, deletedAt, deletedAt, nil},
		{"empty ID", "", createdAt, deletedAt, deletedAt, ErrIDEmpty},
		{"zero createdAt", "user-1", time.Time{}, deletedAt, deletedAt, ErrCreatedAtZero},
		{"zero updatedAt", "user-1", createdAt, time.Time{}, deletedAt, ErrUpdatedAtZero},
		{"updated before created", "user-1", createdAt, createdAt.Add(-time.Hour), deletedAt, ErrInvalidTimestamp},
		{"deleted before created", "user-1", createdAt, deletedAt, createdAt.Add(-time.Hour), ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := RestoreDeleted(tt.id, tt.createdAt, tt.updatedAt, tt.deletedAt)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.True(t, u.IsDeleted())
			require.Equal(t, DeletedDisplayName, u.DisplayName())
			require.Empty(t, u.Email().String())
		})
	}
}
