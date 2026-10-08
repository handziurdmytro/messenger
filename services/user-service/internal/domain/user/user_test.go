package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testTime = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

func newTestUser(t *testing.T) User {
	t.Helper()
	u, err := New(ID("user-1"), "user@gmail.com", testTime)
	require.NoError(t, err)
	return u
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		id        ID
		email     string
		createdAt time.Time
		wantEmail string
		wantErr   error
	}{
		{"valid", "user-1", "user@gmail.com", testTime, "user@gmail.com", nil},
		{"trim email", "user-1", "  user@gmail.com  ", testTime, "user@gmail.com", nil},
		{"empty id", "", "user@gmail.com", testTime, "", ErrIDEmpty},
		{"empty email", "user-1", "", testTime, "", ErrEmailEmpty},
		{"invalid email", "user-1", "invalid", testTime, "", ErrEmailInvalid},
		{"zero created at", "user-1", "user@gmail.com", time.Time{}, "", ErrCreatedAtZero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := New(tt.id, tt.email, tt.createdAt)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, User{}, u)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.id, u.ID())
			require.Equal(t, tt.wantEmail, u.Email().String())
			require.Equal(t, "user", u.DisplayName())
			require.Empty(t, u.AvatarURL())
			require.Equal(t, tt.createdAt, u.CreatedAt())
			require.Equal(t, tt.createdAt, u.UpdatedAt())
		})
	}
}

func TestRestore(t *testing.T) {
	updatedAt := testTime.Add(time.Hour)
	tests := []struct {
		name        string
		id          ID
		email       string
		displayName string
		createdAt   time.Time
		updatedAt   time.Time
		wantErr     error
	}{
		{"valid", "user-1", "user@gmail.com", "Custom name", testTime, updatedAt, nil},
		{"empty id", "", "user@gmail.com", "Custom name", testTime, updatedAt, ErrIDEmpty},
		{"empty email", "user-1", "", "Custom name", testTime, updatedAt, ErrEmailEmpty},
		{"invalid email", "user-1", "invalid", "Custom name", testTime, updatedAt, ErrEmailInvalid},
		{"empty name", "user-1", "user@gmail.com", "   ", testTime, updatedAt, ErrDisplayNameEmpty},
		{"zero created at", "user-1", "user@gmail.com", "Custom name", time.Time{}, updatedAt, ErrCreatedAtZero},
		{"zero updated at", "user-1", "user@gmail.com", "Custom name", testTime, time.Time{}, ErrUpdatedAtZero},
		{"updated before created", "user-1", "user@gmail.com", "Custom name", testTime, testTime.Add(-time.Second), ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := Restore(tt.id, tt.email, tt.displayName, "avatar.png", tt.createdAt, tt.updatedAt)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, User{}, u)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.id, u.ID())
			require.Equal(t, tt.email, u.Email().String())
			require.Equal(t, tt.displayName, u.DisplayName())
			require.Equal(t, "avatar.png", u.AvatarURL())
			require.Equal(t, tt.createdAt, u.CreatedAt())
			require.Equal(t, tt.updatedAt, u.UpdatedAt())
		})
	}
}

func TestUser_ChangeDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		now     time.Time
		want    string
		wantErr error
	}{
		{"valid", "Dmytro", testTime.Add(time.Minute), "Dmytro", nil},
		{"trim spaces", "  Dmytro  ", testTime.Add(time.Minute), "Dmytro", nil},
		{"empty name", "", testTime.Add(time.Minute), "", ErrDisplayNameEmpty},
		{"spaces only", "   ", testTime.Add(time.Minute), "", ErrDisplayNameEmpty},
		{"zero time", "Dmytro", time.Time{}, "", ErrInvalidTimestamp},
		{"time before creation", "Dmytro", testTime.Add(-time.Second), "", ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newTestUser(t)
			before := u
			err := u.ChangeDisplayName(tt.input, tt.now)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, before, u, "failed mutation must not change user")
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, u.DisplayName())
			require.Equal(t, tt.now, u.UpdatedAt())
			require.Equal(t, before.ID(), u.ID())
			require.Equal(t, before.Email(), u.Email())
			require.Equal(t, before.CreatedAt(), u.CreatedAt())
			require.Equal(t, before.AvatarURL(), u.AvatarURL())
		})
	}
}

func TestUser_ChangeAvatarURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		now     time.Time
		want    string
		wantErr error
	}{
		{"valid", "https://example.com/avatar.png", testTime.Add(time.Minute), "https://example.com/avatar.png", nil},
		{"trim spaces", "  avatar.png  ", testTime.Add(time.Minute), "avatar.png", nil},
		{"clear avatar", "", testTime.Add(time.Minute), "", nil},
		{"zero time", "avatar.png", time.Time{}, "", ErrInvalidTimestamp},
		{"time before creation", "avatar.png", testTime.Add(-time.Second), "", ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newTestUser(t)
			before := u
			err := u.ChangeAvatarURL(tt.input, tt.now)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, before, u, "failed mutation must not change user")
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, u.AvatarURL())
			require.Equal(t, tt.now, u.UpdatedAt())
			require.Equal(t, before.ID(), u.ID())
			require.Equal(t, before.Email(), u.Email())
			require.Equal(t, before.CreatedAt(), u.CreatedAt())
			require.Equal(t, before.DisplayName(), u.DisplayName())
		})
	}
}
