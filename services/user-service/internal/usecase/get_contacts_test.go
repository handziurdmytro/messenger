package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
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
					GetByUserID(mock.Anything, user.ID("owner-1"), outbound.Page{Limit: 100}).
					Return([]user.User{u}, nil).
					Once()
			},
		},
		{
			name: "empty list",
			want: []user.User{},
			setupMocks: func(contacts *MockContactRepository) {
				contacts.EXPECT().
					GetByUserID(mock.Anything, user.ID("owner-1"), outbound.Page{Limit: 100}).
					Return([]user.User{}, nil).
					Once()
			},
		},
		{
			name:    "repository error",
			wantErr: outbound.ErrPersistence,
			setupMocks: func(contacts *MockContactRepository) {
				contacts.EXPECT().
					GetByUserID(mock.Anything, user.ID("owner-1"), outbound.Page{Limit: 100}).
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
			got, err := uc.Execute(context.Background(), "owner-1", 0, 0)

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

func TestGetContactsUseCase_ExecutePage(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		offset   int
		wantPage outbound.Page
	}{
		{"default limit", 0, 0, outbound.Page{Limit: defaultContactsLimit}},
		{"custom page", 10, 20, outbound.Page{Limit: 10, Offset: 20}},
		{"too large limit", maxContactsLimit + 1, 0, outbound.Page{Limit: defaultContactsLimit}},
		{"negative offset", 10, -1, outbound.Page{Limit: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contacts := NewMockContactRepository(t)
			contacts.EXPECT().GetByUserID(mock.Anything, user.ID("owner-1"), tt.wantPage).Return([]user.User{}, nil).Once()

			_, err := NewGetContactsUseCase(contacts).Execute(context.Background(), "owner-1", tt.limit, tt.offset)
			require.NoError(t, err)
		})
	}
}
