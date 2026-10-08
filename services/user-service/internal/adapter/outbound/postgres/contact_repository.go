package postgres

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const contactsPrimaryKeyConstraint = "contacts_pkey"

type ContactRepository struct {
	q *db.Queries
}

func NewContactRepository(q *db.Queries) *ContactRepository {
	return &ContactRepository{q: q}
}

func (r *ContactRepository) Add(ctx context.Context, c contact.Contact) error {
	err := r.q.AddContact(ctx, db.AddContactParams{
		UserID: c.UserID().String(), ContactID: c.ContactID().String(), CreatedAt: c.CreatedAt(),
	})
	if err == nil {
		return nil
	}

	if pgkit.IsConstraint(err, pgkit.CodeUniqueViolation, contactsPrimaryKeyConstraint) {
		return contact.ErrAlreadyExists
	}

	return errkit.Wrap(outbound.ErrPersistence, "add contact", err)
}

func (r *ContactRepository) GetByUserID(ctx context.Context, userID user.ID) ([]user.User, error) {
	rows, err := r.q.GetContactsByUserID(ctx, userID.String())
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "get contacts", err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		u, err := toDomainUser(row)
		if err != nil {
			return nil, fmt.Errorf("restore contact user %s: %w", row.ID, err)
		}
		users = append(users, u)
	}

	return users, nil
}
