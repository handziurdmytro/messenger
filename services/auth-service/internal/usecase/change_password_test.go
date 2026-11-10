package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

func TestChangePasswordUseCase_Execute(t *testing.T) {
	const (
		current = "current-pass"
		next    = "next-password"
	)

	user, err := auth.NewUser("user@example.com", current)
	require.NoError(t, err)

	newHash := mock.MatchedBy(func(hash auth.PasswordHash) bool {
		return auth.CheckPassword(next, hash) == nil
	})

	tests := []struct {
		name    string
		input   ChangePasswordInput
		setup   func(*MockUserRepository, *MockTokenRepository)
		wantErr error
	}{
		{
			name:  "updates password and revokes tokens",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: next},
			setup: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().UpdatePassword(mock.Anything, user.ID, newHash).Return(nil).Once()
				tokens.EXPECT().DeleteAllRefresh(mock.Anything, user.ID).Return(nil).Once()
			},
		},
		{
			name:  "user not found",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: next},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(auth.User{}, auth.ErrUserNotFound).Once()
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:  "find user fails",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: next},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(auth.User{}, outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
		{
			name:  "wrong current password",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: "wrong-pass", NewPassword: next},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
			},
			wantErr: auth.ErrPasswordInvalid,
		},
		{
			name:  "new password equals current",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: current},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
			},
			wantErr: auth.ErrPasswordUnchanged,
		},
		{
			name:  "new password too short",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: "short"},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
			},
			wantErr: auth.ErrPasswordTooShort,
		},
		{
			name:  "user disappears during update",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: next},
			setup: func(users *MockUserRepository, _ *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().UpdatePassword(mock.Anything, user.ID, newHash).Return(auth.ErrUserNotFound).Once()
			},
			wantErr: auth.ErrUserNotFound,
		},
		{
			name:  "token revocation fails",
			input: ChangePasswordInput{UserID: user.ID, CurrentPassword: current, NewPassword: next},
			setup: func(users *MockUserRepository, tokens *MockTokenRepository) {
				users.EXPECT().FindByID(mock.Anything, user.ID).Return(user, nil).Once()
				users.EXPECT().UpdatePassword(mock.Anything, user.ID, newHash).Return(nil).Once()
				tokens.EXPECT().DeleteAllRefresh(mock.Anything, user.ID).Return(outbound.ErrPersistence).Once()
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := NewMockUserRepository(t)
			tokens := NewMockTokenRepository(t)
			tt.setup(users, tokens)

			uc := NewChangePasswordUseCase(users, testTransactor(users, tokens, NewMockOutboxRepository(t)))
			err := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
