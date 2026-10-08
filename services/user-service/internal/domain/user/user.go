package user

import (
	"strings"
	"time"
)

type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) IsZero() bool {
	return id == ""
}

type User struct {
	id          ID
	email       Email
	displayName string
	avatarURL   string
	createdAt   time.Time
	updatedAt   time.Time
}

func New(
	id ID,
	rawEmail string,
	createdAt time.Time,
) (User, error) {
	if id.IsZero() {
		return User{}, ErrIDEmpty
	}

	email, err := NewEmail(rawEmail)
	if err != nil {
		return User{}, err
	}

	if createdAt.IsZero() {
		return User{}, ErrCreatedAtZero
	}

	return User{
		id:          id,
		email:       email,
		displayName: defaultDisplayName(email),
		createdAt:   createdAt,
		updatedAt:   createdAt,
	}, nil
}

func Restore(
	id ID,
	rawEmail string,
	displayName string,
	avatarURL string,
	createdAt time.Time,
	updatedAt time.Time,
) (User, error) {
	if id.IsZero() {
		return User{}, ErrIDEmpty
	}

	email, err := NewEmail(rawEmail)
	if err != nil {
		return User{}, err
	}

	if strings.TrimSpace(displayName) == "" {
		return User{}, ErrDisplayNameEmpty
	}

	if createdAt.IsZero() {
		return User{}, ErrCreatedAtZero
	}

	if updatedAt.IsZero() {
		return User{}, ErrUpdatedAtZero
	}

	if updatedAt.Before(createdAt) {
		return User{}, ErrInvalidTimestamp
	}

	return User{
		id:          id,
		email:       email,
		displayName: displayName,
		avatarURL:   avatarURL,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (u *User) ChangeDisplayName(
	displayName string,
	now time.Time,
) error {
	displayName = strings.TrimSpace(displayName)

	if displayName == "" {
		return ErrDisplayNameEmpty
	}

	if now.IsZero() || now.Before(u.createdAt) {
		return ErrInvalidTimestamp
	}

	u.displayName = displayName
	u.updatedAt = now

	return nil
}

func (u *User) ChangeAvatarURL(
	avatarURL string,
	now time.Time,
) error {
	if now.IsZero() || now.Before(u.createdAt) {
		return ErrInvalidTimestamp
	}

	u.avatarURL = strings.TrimSpace(avatarURL)
	u.updatedAt = now

	return nil
}

func (u *User) ID() ID {
	return u.id
}

func (u *User) Email() Email {
	return u.email
}

func (u *User) DisplayName() string {
	return u.displayName
}

func (u *User) AvatarURL() string {
	return u.avatarURL
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func defaultDisplayName(email Email) string {
	if before, _, ok := strings.Cut(email.String(), "@"); ok && before != "" {
		return before
	}

	return email.String()
}
