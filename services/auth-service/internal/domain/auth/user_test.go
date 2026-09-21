package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		plainPassword string
		wantErr       error
	}{
		{"success", "user@gmail.com", "pass1234", nil},
		{"empty email", "", "pass1234", ErrEmailEmpty},
		{"invalid email", "invalid-email-format", "pass1234", ErrEmailInvalid},
		{"short password", "user@gmail.com", "1234567", ErrPasswordTooShort},
		{"long password", "user@gmail.com", strings.Repeat("a", 73), ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			user, err := NewUser(tt.email, tt.plainPassword)
			after := time.Now()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, User{}, user)
				return
			}

			require.NoError(t, err)
			require.NotEqual(t, User{}, user)
			require.NotEmpty(t, user.ID)

			assert.Equal(t, tt.email, user.Email)
			assert.WithinRange(t, user.CreatedAt, before, after)
			assert.NotEqual(t, tt.plainPassword, string(user.PasswordHash))
			assert.NoError(t, CheckPassword(tt.plainPassword, user.PasswordHash))
		})
	}
}
