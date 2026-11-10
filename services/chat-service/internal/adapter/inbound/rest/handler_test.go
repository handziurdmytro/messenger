package rest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
)

var errInternal = errors.New("database unavailable")

func serve(t *testing.T, services Services, method, target, body string, userID chat.UserID) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if userID != "" {
		req.Header.Set("X-User-ID", string(userID))
	}

	rec := httptest.NewRecorder()
	NewServer("0", services, nil, slog.New(slog.NewTextHandler(io.Discard, nil))).http.Handler.ServeHTTP(rec, req)
	return rec
}

func newUser() chat.UserID {
	return chat.UserID(uuid.NewString())
}

func TestHandler_Unauthorized(t *testing.T) {
	id := chat.NewID()
	routes := []struct{ method, target string }{
		{http.MethodGet, "/chats"},
		{http.MethodPost, "/chats"},
		{http.MethodGet, "/chats/" + string(id)},
		{http.MethodDelete, "/chats/" + string(id)},
		{http.MethodGet, "/chats/" + string(id) + "/participants"},
		{http.MethodPost, "/chats/groups"},
		{http.MethodPatch, "/chats/" + string(id)},
		{http.MethodDelete, "/chats/" + string(id) + "/participants/" + string(newUser())},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.target, func(t *testing.T) {
			rec := serve(t, Services{}, route.method, route.target, "", "invalid")
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestHandler_ListChats(t *testing.T) {
	owner := newUser()
	c, err := chat.NewPrivate(owner, newUser())
	require.NoError(t, err)
	next := &usecase.ChatCursor{CreatedAt: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), ID: c.ID()}

	t.Run("returns chats and next cursor", func(t *testing.T) {
		uc := NewMockListChatsUseCase(t)
		uc.EXPECT().Execute(mock.Anything, owner, 0, (*usecase.ChatCursor)(nil)).Return(usecase.ChatList{Chats: []chat.Chat{c}, Next: next}, nil).Once()

		rec := serve(t, Services{ListChats: uc}, http.MethodGet, "/chats", "", owner)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), string(c.ID()))

		cursor, err := decodeCursor(rec.Header().Get(headerNextCursor))
		require.NoError(t, err)
		require.Equal(t, *next, cursor)
	})

	t.Run("passes limit and cursor", func(t *testing.T) {
		uc := NewMockListChatsUseCase(t)
		uc.EXPECT().Execute(mock.Anything, owner, 5, next).Return(usecase.ChatList{}, nil).Once()

		rec := serve(t, Services{ListChats: uc}, http.MethodGet, "/chats?limit=5&before="+encodeCursor(*next), "", owner)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get(headerNextCursor))
		require.JSONEq(t, `[]`, rec.Body.String())
	})

	for _, query := range []string{"limit=0", "limit=abc", "before=***", "before=" + "bm90LWEtY3Vyc29y"} {
		t.Run("rejects "+query, func(t *testing.T) {
			rec := serve(t, Services{}, http.MethodGet, "/chats?"+query, "", owner)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestHandler_CreateChat(t *testing.T) {
	alice, bob := newUser(), newUser()
	c, err := chat.NewPrivate(alice, bob)
	require.NoError(t, err)

	tests := []struct {
		name       string
		body       string
		ucErr      error
		callUC     bool
		wantStatus int
	}{
		{name: "success", body: `{"contact_id":"` + string(bob) + `"}`, callUC: true, wantStatus: http.StatusOK},
		{name: "blocked", body: `{"contact_id":"` + string(bob) + `"}`, callUC: true, ucErr: usecase.ErrBlocked, wantStatus: http.StatusForbidden},
		{name: "internal error", body: `{"contact_id":"` + string(bob) + `"}`, callUC: true, ucErr: errInternal, wantStatus: http.StatusInternalServerError},
		{name: "invalid contact", body: `{"contact_id":"bad"}`, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"contact_id":"` + string(bob) + `","x":1}`, wantStatus: http.StatusBadRequest},
		{name: "trailing data", body: `{"contact_id":"` + string(bob) + `"}{}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockCreateChatUseCase(t)
			if tt.callUC {
				uc.EXPECT().Execute(mock.Anything, alice, bob).Return(c, tt.ucErr).Once()
			}

			rec := serve(t, Services{CreateChat: uc}, http.MethodPost, "/chats", tt.body, alice)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_GetChatAndParticipants(t *testing.T) {
	owner, member := newUser(), newUser()
	group, err := chat.NewGroup(owner, "Team", []chat.UserID{member})
	require.NoError(t, err)

	t.Run("get chat", func(t *testing.T) {
		uc := NewMockGetChatUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), owner).Return(group, nil).Once()

		rec := serve(t, Services{GetChat: uc}, http.MethodGet, "/chats/"+string(group.ID()), "", owner)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"title":"Team"`)
		require.Contains(t, rec.Body.String(), `"owner_id":"`+string(owner)+`"`)
	})

	t.Run("participants with roles", func(t *testing.T) {
		uc := NewMockGetChatUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), member).Return(group, nil).Once()

		rec := serve(t, Services{GetChat: uc}, http.MethodGet, "/chats/"+string(group.ID())+"/participants", "", member)
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `[{"user_id":"`+string(owner)+`","role":"owner"},{"user_id":"`+string(member)+`","role":"member"}]`, rec.Body.String())
	})

	errorCases := []struct {
		err        error
		wantStatus int
	}{
		{usecase.ErrChatNotFound, http.StatusNotFound},
		{usecase.ErrForbidden, http.StatusForbidden},
		{chat.ErrInvalidChatID, http.StatusBadRequest},
		{errInternal, http.StatusInternalServerError},
	}
	for _, tc := range errorCases {
		t.Run("maps "+tc.err.Error(), func(t *testing.T) {
			uc := NewMockGetChatUseCase(t)
			uc.EXPECT().Execute(mock.Anything, group.ID(), owner).Return(chat.Chat{}, tc.err).Once()

			rec := serve(t, Services{GetChat: uc}, http.MethodGet, "/chats/"+string(group.ID()), "", owner)
			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}

	t.Run("invalid chat ID", func(t *testing.T) {
		rec := serve(t, Services{}, http.MethodGet, "/chats/bad/participants", "", owner)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_DeleteChat(t *testing.T) {
	user := newUser()
	id := chat.NewID()

	tests := []struct {
		err        error
		wantStatus int
	}{
		{nil, http.StatusNoContent},
		{usecase.ErrForbidden, http.StatusForbidden},
		{chat.ErrUnsupportedChatType, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.wantStatus), func(t *testing.T) {
			uc := NewMockDeleteChatUseCase(t)
			uc.EXPECT().Execute(mock.Anything, id, user).Return(tt.err).Once()

			rec := serve(t, Services{DeleteChat: uc}, http.MethodDelete, "/chats/"+string(id), "", user)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_Groups(t *testing.T) {
	owner, member := newUser(), newUser()
	group, err := chat.NewGroup(owner, "Team", []chat.UserID{member})
	require.NoError(t, err)
	groupPath := "/chats/" + string(group.ID())
	anyChange := mock.AnythingOfType("usecase.GroupChange")

	t.Run("create group", func(t *testing.T) {
		uc := NewMockCreateGroupUseCase(t)
		uc.EXPECT().Execute(mock.Anything, usecase.CreateGroupInput{Owner: owner, Title: "Team", Members: []chat.UserID{member}}).Return(group, nil).Once()

		rec := serve(t, Services{CreateGroup: uc}, http.MethodPost, "/chats/groups", `{"title":"Team","member_ids":["`+string(member)+`"]}`, owner)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"type":"group"`)
	})

	t.Run("create group invalid body", func(t *testing.T) {
		rec := serve(t, Services{}, http.MethodPost, "/chats/groups", `{"title":1}`, owner)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rename applies change", func(t *testing.T) {
		uc := NewMockUpdateGroupUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), owner, anyChange).RunAndReturn(
			func(_ context.Context, _ chat.ID, by chat.UserID, change usecase.GroupChange) (chat.Chat, error) {
				c := group
				return c, change(&c, by)
			}).Once()

		rec := serve(t, Services{UpdateGroup: uc}, http.MethodPatch, groupPath, `{"title":"Renamed"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"title":"Renamed"`)
	})

	t.Run("add participants", func(t *testing.T) {
		uc := NewMockUpdateGroupUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), owner, anyChange).Return(group, nil).Once()

		rec := serve(t, Services{UpdateGroup: uc}, http.MethodPost, groupPath+"/participants", `{"user_ids":["`+string(newUser())+`"]}`, owner)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("remove participant", func(t *testing.T) {
		uc := NewMockUpdateGroupUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), owner, anyChange).Return(group, nil).Once()

		rec := serve(t, Services{UpdateGroup: uc}, http.MethodDelete, groupPath+"/participants/"+string(member), "", owner)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("removing self leaves the chat", func(t *testing.T) {
		uc := NewMockDeleteChatUseCase(t)
		uc.EXPECT().Execute(mock.Anything, group.ID(), member).Return(nil).Once()

		rec := serve(t, Services{DeleteChat: uc}, http.MethodDelete, groupPath+"/participants/"+string(member), "", member)
		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("invalid participant ID", func(t *testing.T) {
		rec := serve(t, Services{}, http.MethodDelete, groupPath+"/participants/bad", "", owner)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	errorCases := []struct {
		err        error
		wantStatus int
	}{
		{chat.ErrInvalidTitle, http.StatusBadRequest},
		{chat.ErrTooManyParticipants, http.StatusBadRequest},
		{chat.ErrCannotRemoveOwner, http.StatusBadRequest},
		{chat.ErrNotOwner, http.StatusForbidden},
		{chat.ErrNotParticipant, http.StatusNotFound},
		{chat.ErrAlreadyParticipant, http.StatusConflict},
		{chat.ErrNotGroup, http.StatusUnprocessableEntity},
	}
	for _, tc := range errorCases {
		t.Run("maps "+tc.err.Error(), func(t *testing.T) {
			uc := NewMockUpdateGroupUseCase(t)
			uc.EXPECT().Execute(mock.Anything, group.ID(), owner, anyChange).Return(chat.Chat{}, tc.err).Once()

			rec := serve(t, Services{UpdateGroup: uc}, http.MethodPatch, groupPath, `{"title":"x"}`, owner)
			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
