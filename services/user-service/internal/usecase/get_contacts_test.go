package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetContactsUseCase_Execute(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	u, err := user.New("contact-1", "contact@gmail.com", now)
	require.NoError(t, err)

	tests := []struct {
		name       string
		want       []user.User
		wantErr    error
		setupMocks func(*MockContactRepository)
	}{
		{
			name: "success",
			want: []user.User{u},
			setupMocks: func(contacts *MockContactRepository) {
				contacts.EXPECT().
					GetByUserID(mock.Anything, user.ID("owner-1")).
					Return([]user.User{u}, nil).
					Once()
			},
		},
		{
			name: "empty list",
			want: []user.User{},
			setupMocks: func(contacts *MockContactRepository) {
				contacts.EXPECT().
					GetByUserID(mock.Anything, user.ID("owner-1")).
					Return([]user.User{}, nil).
					Once()
			},
		},
		{
			name:    "repository error",
			wantErr: outbound.ErrPersistence,
			setupMocks: func(contacts *MockContactRepository) {
				contacts.EXPECT().
					GetByUserID(mock.Anything, user.ID("owner-1")).
					Return(nil, outbound.ErrPersistence).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contacts := NewMockContactRepository(t)
			tt.setupMocks(contacts)

			uc := NewGetContactsUseCase(contacts)
			got, err := uc.Execute(context.Background(), "owner-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
