package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name    string
		plain   string
		wantErr error
	}{
		{"success", "pass1234", nil},
		{"short password", "123", ErrPasswordTooShort},
		{"long password", strings.Repeat("a", 73), ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashPassword(tt.plain)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	validPassword := "pass1234"
	validHash, err := HashPassword(validPassword)
	if err != nil {
		t.Fatalf("HashPassword() setup error = %v, wantErr nil", err)
	}

	tests := []struct {
		name    string
		plain   string
		hash    PasswordHash
		wantErr error
	}{
		{"success", validPassword, validHash, nil},
		{"invalid password", "wrongpass", validHash, ErrPasswordInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPassword(tt.plain, tt.hash)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CheckPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
