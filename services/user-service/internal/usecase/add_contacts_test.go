package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAddContactUseCase_Execute(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	target, err := user.New("user-2", "target@gmail.com", now)
	require.NoError(t, err)

	validContact := mock.MatchedBy(func(c contact.Contact) bool {
		return c.UserID() == "user-1" && c.ContactID() == "user-2" && !c.CreatedAt().IsZero()
	})

	tests := []struct {
		name       string
		userID     user.ID
		contactID  user.ID
		wantErr    error
		setupMocks func(*MockUserRepository, *MockContactRepository)
	}{
		{
			name:      "success",
			userID:    "user-1",
			contactID: "user-2",
			setupMocks: func(users *MockUserRepository, contacts *MockContactRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("user-2")).
					Return(target, nil).
					Once()

				contacts.EXPECT().
					Add(mock.Anything, validContact).
					Return(nil).
					Once()
			},
		},
		{
			name:       "cannot add self",
			userID:     "user-1",
			contactID:  "user-1",
			wantErr:    contact.ErrCannotAddSelf,
			setupMocks: func(_ *MockUserRepository, _ *MockContactRepository) {},
		},
		{
			name:       "empty user ID",
			contactID:  "user-2",
			wantErr:    contact.ErrUserIDEmpty,
			setupMocks: func(_ *MockUserRepository, _ *MockContactRepository) {},
		},
		{
			name:       "empty contact ID",
			userID:     "user-1",
			wantErr:    contact.ErrContactIDEmpty,
			setupMocks: func(_ *MockUserRepository, _ *MockContactRepository) {},
		},
		{
			name:      "target not found",
			userID:    "user-1",
			contactID: "user-2",
			wantErr:   user.ErrNotFound,
			setupMocks: func(users *MockUserRepository, contacts *MockContactRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("user-2")).
					Return(user.User{}, user.ErrNotFound).
					Once()
			},
		},
		{
			name:      "user repository error",
			userID:    "user-1",
			contactID: "user-2",
			wantErr:   outbound.ErrPersistence,
			setupMocks: func(users *MockUserRepository, contacts *MockContactRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("user-2")).
					Return(user.User{}, outbound.ErrPersistence).
					Once()
			},
		},
		{
			name:      "already exists",
			userID:    "user-1",
			contactID: "user-2",
			wantErr:   contact.ErrAlreadyExists,
			setupMocks: func(users *MockUserRepository, contacts *MockContactRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("user-2")).
					Return(target, nil).
					Once()

				contacts.EXPECT().
					Add(mock.Anything, validContact).
					Return(contact.ErrAlreadyExists).
					Once()
			},
		},
		{
			name:      "contact repository error",
			userID:    "user-1",
			contactID: "user-2",
			wantErr:   outbound.ErrPersistence,
			setupMocks: func(users *MockUserRepository, contacts *MockContactRepository) {
				users.EXPECT().
					FindByID(mock.Anything, user.ID("user-2")).
					Return(target, nil).
					Once()

				contacts.EXPECT().
					Add(mock.Anything, validContact).
					Return(outbound.ErrPersistence).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			contacts := NewMockContactRepository(t)
			tt.setupMocks(users, contacts)

			uc := NewAddContactUseCase(users, contacts)
			err := uc.Execute(context.Background(), tt.userID, tt.contactID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
