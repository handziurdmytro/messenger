package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func TestGetProfilesUseCase_Execute(t *testing.T) {
	alice, err := user.New("user-1", "alice@example.com", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	tooMany := make([]user.ID, 0, MaxProfilesPerRequest+1)
	for i := range MaxProfilesPerRequest + 1 {
		tooMany = append(tooMany, user.ID(fmt.Sprintf("user-%d", i)))
	}

	tests := []struct {
		name    string
		ids     []user.ID
		setup   func(*MockUserRepository)
		want    []user.User
		wantErr error
	}{
		{
			name: "deduplicates and skips empty IDs",
			ids:  []user.ID{"user-1", "", "user-2", "user-1"},
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByIDs(mock.Anything, []user.ID{"user-1", "user-2"}).Return([]user.User{alice}, nil).Once()
			},
			want: []user.User{alice},
		},
		{
			name:  "no IDs",
			ids:   []user.ID{"", ""},
			setup: func(_ *MockUserRepository) {},
			want:  []user.User{},
		},
		{
			name:    "too many IDs",
			ids:     tooMany,
			setup:   func(_ *MockUserRepository) {},
			wantErr: ErrTooManyIDs,
		},
		{
			name: "repository error",
			ids:  []user.ID{"user-1"},
			setup: func(users *MockUserRepository) {
				users.EXPECT().FindByIDs(mock.Anything, []user.ID{"user-1"}).Return(nil, outbound.ErrPersistence).Once()
			},
			wantErr: outbound.ErrPersistence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tt.setup(users)

			got, err := NewGetProfilesUseCase(users).Execute(context.Background(), tt.ids)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
