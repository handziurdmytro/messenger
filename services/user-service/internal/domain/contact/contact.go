package contact

import (
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
)

type Contact struct {
	userID    user.ID
	contactID user.ID
	createdAt time.Time
}

func New(
	userID user.ID,
	contactID user.ID,
	createdAt time.Time,
) (Contact, error) {
	if userID.IsZero() {
		return Contact{}, ErrUserIDEmpty
	}

	if contactID.IsZero() {
		return Contact{}, ErrContactIDEmpty
	}

	if userID == contactID {
		return Contact{}, ErrCannotAddSelf
	}

	if createdAt.IsZero() {
		return Contact{}, ErrCreatedAtZero
	}

	return Contact{
		userID:    userID,
		contactID: contactID,
		createdAt: createdAt,
	}, nil
}

func Restore(
	userID user.ID,
	contactID user.ID,
	createdAt time.Time,
) (Contact, error) {
	return New(userID, contactID, createdAt)
}

func (c Contact) UserID() user.ID {
	return c.userID
}

func (c Contact) ContactID() user.ID {
	return c.contactID
}

func (c Contact) CreatedAt() time.Time {
	return c.createdAt
}
