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

func TestUpdateProfileUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	newUser := func(t *testing.T, at time.Time) user.User {
		t.Helper()
		u, err := user.New("user-1", "test@gmail.com", at)
		require.NoError(t, err)
		return u
	}

	expectFind := func(users *MockUserRepository, u user.User) {
		users.EXPECT().
			FindByID(mock.Anything, user.ID("user-1")).
			Return(u, nil).
			Once()
	}

	t.Run("change display name", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		users.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(u user.User) bool {
				return u.ID() == original.ID() && u.Email() == original.Email() &&
					u.DisplayName() == "Dmytro" && u.AvatarURL() == original.AvatarURL() &&
					u.CreatedAt().Equal(createdAt) && u.UpdatedAt().After(createdAt)
			})).
			Return(nil).
			Once()

		name := "  Dmytro  "
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{DisplayName: &name})

		require.NoError(t, err)
		require.Equal(t, "Dmytro", got.DisplayName())
		require.Empty(t, got.AvatarURL())
		require.True(t, got.UpdatedAt().After(createdAt))
	})

	t.Run("change avatar", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		users.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(u user.User) bool {
				return u.ID() == original.ID() && u.Email() == original.Email() &&
					u.DisplayName() == original.DisplayName() &&
					u.AvatarURL() == "https://example.com/avatar.png" &&
					u.CreatedAt().Equal(createdAt) && u.UpdatedAt().After(createdAt)
			})).
			Return(nil).
			Once()

		avatar := "  https://example.com/avatar.png  "
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{AvatarURL: &avatar})

		require.NoError(t, err)
		require.Equal(t, original.DisplayName(), got.DisplayName())
		require.Equal(t, "https://example.com/avatar.png", got.AvatarURL())
		require.True(t, got.UpdatedAt().After(createdAt))
	})

	t.Run("change both", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		users.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(u user.User) bool {
				return u.ID() == original.ID() && u.Email() == original.Email() &&
					u.DisplayName() == "Dmytro" &&
					u.AvatarURL() == "https://example.com/avatar.png" &&
					u.CreatedAt().Equal(createdAt) && u.UpdatedAt().After(createdAt)
			})).
			Return(nil).
			Once()

		name, avatar := "Dmytro", "https://example.com/avatar.png"
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{DisplayName: &name, AvatarURL: &avatar})

		require.NoError(t, err)
		require.Equal(t, "Dmytro", got.DisplayName())
		require.Equal(t, avatar, got.AvatarURL())
		require.True(t, got.UpdatedAt().After(createdAt))
	})

	t.Run("no changes", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{})

		require.NoError(t, err)
		require.Equal(t, original, got)
	})

	t.Run("invalid display name", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		name := "   "
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{DisplayName: &name})

		require.ErrorIs(t, err, user.ErrDisplayNameEmpty)
		require.Equal(t, user.User{}, got)
	})

	t.Run("invalid display name with avatar", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		name, avatar := "   ", "https://example.com/avatar.png"
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{DisplayName: &name, AvatarURL: &avatar})

		require.ErrorIs(t, err, user.ErrDisplayNameEmpty)
		require.Equal(t, user.User{}, got)
	})

	t.Run("invalid avatar timestamp", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		original := newUser(t, future)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		avatar := "https://example.com/avatar.png"
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{AvatarURL: &avatar})

		require.ErrorIs(t, err, user.ErrInvalidTimestamp)
		require.Equal(t, user.User{}, got)
	})

	t.Run("user not found", func(t *testing.T) {
		users := NewMockUserRepository(t)
		users.EXPECT().
			FindByID(mock.Anything, user.ID("user-1")).
			Return(user.User{}, user.ErrNotFound).
			Once()

		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{})

		require.ErrorIs(t, err, user.ErrNotFound)
		require.Equal(t, user.User{}, got)
	})

	t.Run("find repository error", func(t *testing.T) {
		users := NewMockUserRepository(t)
		users.EXPECT().
			FindByID(mock.Anything, user.ID("user-1")).
			Return(user.User{}, outbound.ErrPersistence).
			Once()

		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{})

		require.ErrorIs(t, err, outbound.ErrPersistence)
		require.Equal(t, user.User{}, got)
	})

	t.Run("update repository error", func(t *testing.T) {
		original := newUser(t, createdAt)
		users := NewMockUserRepository(t)
		expectFind(users, original)

		users.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(u user.User) bool {
				return u.DisplayName() == "Dmytro" && u.UpdatedAt().After(createdAt)
			})).
			Return(outbound.ErrPersistence).
			Once()

		name := "Dmytro"
		uc := NewUpdateProfileUseCase(users)
		got, err := uc.Execute(ctx, "user-1", UpdateProfileInput{DisplayName: &name})

		require.ErrorIs(t, err, outbound.ErrPersistence)
		require.Equal(t, user.User{}, got)
	})
}
