package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name    string
		plain   string
		wantErr error
	}{
		{"success", "pass123456", nil},
		{"shortest valid", "pass1234", nil},
		{"longest valid", strings.Repeat("a", 72), nil},
		{"short password", "1234567", ErrPasswordTooShort},
		{"long password", strings.Repeat("a", 73), ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.plain)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, hash)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			assert.NotEqual(t, tt.plain, string(hash))
			assert.NoError(t, CheckPassword(tt.plain, hash))
		})
	}
}

func TestCheckPassword(t *testing.T) {
	validPassword := "pass1234"
	validHash, err := HashPassword(validPassword)
	require.NoError(t, err)

	tests := []struct {
		name    string
		plain   string
		hash    PasswordHash
		wantErr error
	}{
		{"success", validPassword, validHash, nil},
		{"invalid password", "wrongpass", validHash, ErrPasswordInvalid},
		{"invalid hash", validPassword, PasswordHash("garbage"), ErrPasswordInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPassword(tt.plain, tt.hash)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
