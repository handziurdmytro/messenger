package user

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "valid email",
			input: "abc@gmail.com",
			want:  "abc@gmail.com",
		},
		{
			name:  "trim spaces",
			input: "  abc@gmail.com  ",
			want:  "abc@gmail.com",
		},
		{
			name:    "empty email",
			input:   "",
			wantErr: ErrEmailEmpty,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrEmailEmpty,
		},
		{
			name:    "missing at sign",
			input:   "abcgmail.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "missing domain",
			input:   "abc@",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "display name not allowed",
			input:   "John <john@gmail.com>",
			wantErr: ErrEmailInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, email.String())
		})
	}
}
