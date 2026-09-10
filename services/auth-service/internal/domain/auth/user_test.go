package auth

import (
	"errors"
	"testing"
	"time"
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
		{"short password", "user@gmail.com", "123", ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.plainPassword)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if user.ID == "" {
					t.Errorf("NewUser().ID = %v, want non-empty", user.ID)
				}
				if user.Email != tt.email {
					t.Errorf("NewUser().Email = %v, want %v", user.Email, tt.email)
				}
				if time.Since(user.CreatedAt) > time.Second {
					t.Errorf("NewUser().CreatedAt = %v, want recent time", user.CreatedAt)
				}
			}
		})
	}
}
