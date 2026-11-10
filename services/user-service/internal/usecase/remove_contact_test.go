package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func TestRemoveContactUseCase_Execute(t *testing.T) {
	tests := []struct {
		name      string
		userID    user.ID
		contactID user.ID
		repoErr   error
		callRepo  bool
		wantErr   error
	}{
		{name: "success", userID: "user-1", contactID: "user-2", callRepo: true},
		{name: "empty user ID", contactID: "user-2", wantErr: contact.ErrUserIDEmpty},
		{name: "empty contact ID", userID: "user-1", wantErr: contact.ErrContactIDEmpty},
		{name: "not found", userID: "user-1", contactID: "user-2", callRepo: true, repoErr: contact.ErrNotFound, wantErr: contact.ErrNotFound},
		{name: "repository error", userID: "user-1", contactID: "user-2", callRepo: true, repoErr: outbound.ErrPersistence, wantErr: outbound.ErrPersistence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contacts := NewMockContactRepository(t)
			if tt.callRepo {
				contacts.EXPECT().Remove(mock.Anything, tt.userID, tt.contactID).Return(tt.repoErr).Once()
			}

			err := NewRemoveContactUseCase(contacts).Execute(context.Background(), tt.userID, tt.contactID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
