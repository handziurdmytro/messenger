package user

import (
	"net/mail"
	"strings"
)

type Email struct {
	email string
}

func NewEmail(raw string) (Email, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Email{}, ErrEmailEmpty
	}

	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw {
		return Email{}, ErrEmailInvalid
	}

	return Email{raw}, nil
}

func (e Email) String() string {
	return e.email
}
