package usecase

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

var errTest = errors.New("test error")

func newUserID() chat.UserID {
	return chat.UserID(uuid.NewString())
}

func newPrivateChat(t *testing.T, u1, u2 chat.UserID) chat.Chat {
	t.Helper()

	c, err := chat.NewPrivate(u1, u2)
	require.NoError(t, err)
	return c
}

func newGroupChat(t *testing.T, owner chat.UserID, members ...chat.UserID) chat.Chat {
	t.Helper()

	c, err := chat.NewGroup(owner, "Team", members)
	require.NoError(t, err)
	return c
}
