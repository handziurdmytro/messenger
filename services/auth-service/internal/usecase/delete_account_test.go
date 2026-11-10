package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

func TestDeleteAccountUseCase_Execute(t *testing.T) {
	const password = "password1"

	user, err := auth.NewUser("user@example.com", password)
	require.NoError(t, err)

	recentTime := mock.MatchedBy(func(at time.Time) bool {
		return !at.IsZero() && time.Since(at) < time.Minute
	})

	tests := []struct {
		name     string
		password string
		setup    func(*MockUserRepository, *MockOutboxRepository)
		wantErr  error
	}{
		{
			name:     "deletes user and stores event",
			password: password,
			setup: func(users *MockUserRepository, outbox *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().Delete(mock.Anything, user.ID).Return(nil).Once()
				outbox.EXPECT().SaveUserDeleted(mock.Anything, user.ID, recentTime).Return(nil).Once()
			},
		},
		{
			name:     "user not found",
			password: password,
			setup: func(users *MockUserRepository, _ *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(auth.User{}, auth.ErrUserNotFound).Once()
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:     "find user fails",
			password: password,
			setup: func(users *MockUserRepository, _ *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(auth.User{}, outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
		{
			name:     "wrong password",
			password: "wrong-password",
			setup: func(users *MockUserRepository, _ *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
			},
			wantErr: auth.ErrPasswordInvalid,
		},
		{
			name:     "user disappears during delete",
			password: password,
			setup: func(users *MockUserRepository, _ *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().Delete(mock.Anything, user.ID).Return(auth.ErrUserNotFound).Once()
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:     "outbox fails",
			password: password,
			setup: func(users *MockUserRepository, outbox *MockOutboxRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().Delete(mock.Anything, user.ID).Return(nil).Once()
				outbox.EXPECT().SaveUserDeleted(mock.Anything, user.ID, recentTime).Return(outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			outbox := NewMockOutboxRepository(t)
			tt.setup(users, outbox)

			uc := NewDeleteAccountUseCase(users, testTransactor(users, NewMockTokenRepository(t), outbox))
			err := uc.Execute(context.Background(), DeleteAccountInput{UserID: user.ID, Password: tt.password})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
