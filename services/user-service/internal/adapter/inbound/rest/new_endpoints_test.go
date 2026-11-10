package rest

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

var errInternal = errors.New("database unavailable")

func serveRequest(t *testing.T, services Services, method, target, body, userID string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	rec := httptest.NewRecorder()
	server := NewServer("0", services, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.http.Handler.ServeHTTP(rec, req)
	return rec
}

func TestHandler_RemoveContact(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		repoErr    error
		callUC     bool
		wantStatus int
	}{
		{name: "success", userID: "user-1", callUC: true, wantStatus: http.StatusNoContent},
		{name: "not found", userID: "user-1", callUC: true, repoErr: contact.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", userID: "user-1", callUC: true, repoErr: errInternal, wantStatus: http.StatusInternalServerError},
		{name: "missing user", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockRemoveContactUseCase(t)
			if tt.callUC {
				uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(tt.repoErr).Once()
			}

			rec := serveRequest(t, Services{RemoveContact: uc}, http.MethodDelete, "/me/contacts/user-2", "", tt.userID)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_GetProfiles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		alice := testUser(t, "user-1", "alice@example.com")
		uc := NewMockGetProfilesUseCase(t)
		uc.EXPECT().Execute(mock.Anything, []user.ID{"user-1", "user-2"}).Return([]user.User{alice}, nil).Once()

		rec := serveRequest(t, Services{GetProfiles: uc}, http.MethodGet, "/profiles?ids=user-1,%20user-2,", "", "user-1")
		assertJSON(t, rec, http.StatusOK, `[{"id":"user-1","email":"alice@example.com","display_name":"alice","avatar_url":"","deleted":false}]`)
	})

	t.Run("too many IDs", func(t *testing.T) {
		uc := NewMockGetProfilesUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.Anything).Return(nil, usecase.ErrTooManyIDs).Once()

		rec := serveRequest(t, Services{GetProfiles: uc}, http.MethodGet, "/profiles?ids=a", "", "user-1")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		uc := NewMockGetProfilesUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.Anything).Return(nil, errInternal).Once()

		rec := serveRequest(t, Services{GetProfiles: uc}, http.MethodGet, "/profiles?ids=a", "", "user-1")
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHandler_Pagination(t *testing.T) {
	t.Run("contacts pass limit and offset", func(t *testing.T) {
		uc := NewMockGetContactsUseCase(t)
		uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), 5, 10).Return([]user.User{}, nil).Once()

		rec := serveRequest(t, Services{GetContacts: uc}, http.MethodGet, "/me/contacts?limit=5&offset=10", "", "user-1")
		assertJSON(t, rec, http.StatusOK, `[]`)
	})

	t.Run("search uses default limit", func(t *testing.T) {
		uc := NewMockSearchUsersUseCase(t)
		uc.EXPECT().Execute(mock.Anything, "al", defaultSearchLimit, 3).Return([]user.User{}, nil).Once()

		rec := serveRequest(t, Services{SearchUsers: uc}, http.MethodGet, "/search?q=al&offset=3", "", "user-1")
		require.Equal(t, http.StatusOK, rec.Code)
	})

	for _, target := range []string{"/me/contacts?limit=0", "/me/contacts?limit=x", "/me/contacts?offset=-1", "/search?q=a&limit=-2", "/me/blocks?offset=x"} {
		t.Run("rejects "+target, func(t *testing.T) {
			rec := serveRequest(t, Services{}, http.MethodGet, target, "", "user-1")
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestHandler_BlockUser(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		ucErr      error
		callUC     bool
		wantStatus int
	}{
		{name: "success", body: `{"user_id":"user-2"}`, callUC: true, wantStatus: http.StatusNoContent},
		{name: "self", body: `{"user_id":"user-2"}`, callUC: true, ucErr: block.ErrCannotBlockSelf, wantStatus: http.StatusBadRequest},
		{name: "already blocked", body: `{"user_id":"user-2"}`, callUC: true, ucErr: block.ErrAlreadyBlocked, wantStatus: http.StatusConflict},
		{name: "user not found", body: `{"user_id":"user-2"}`, callUC: true, ucErr: user.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", body: `{"user_id":"user-2"}`, callUC: true, ucErr: errInternal, wantStatus: http.StatusInternalServerError},
		{name: "invalid body", body: `{`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockBlockUserUseCase(t)
			if tt.callUC {
				uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(tt.ucErr).Once()
			}

			rec := serveRequest(t, Services{BlockUser: uc}, http.MethodPost, "/me/blocks", tt.body, "user-1")
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_UnblockUser(t *testing.T) {
	tests := []struct {
		name       string
		ucErr      error
		wantStatus int
	}{
		{"success", nil, http.StatusNoContent},
		{"not found", block.ErrNotFound, http.StatusNotFound},
		{"internal error", errInternal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockUnblockUserUseCase(t)
			uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(tt.ucErr).Once()

			rec := serveRequest(t, Services{UnblockUser: uc}, http.MethodDelete, "/me/blocks/user-2", "", "user-1")
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_ListBlocked(t *testing.T) {
	bob := testUser(t, "user-2", "bob@example.com")
	uc := NewMockListBlockedUseCase(t)
	uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), 0, 0).Return([]user.User{bob}, nil).Once()

	rec := serveRequest(t, Services{ListBlocked: uc}, http.MethodGet, "/me/blocks", "", "user-1")
	assertJSON(t, rec, http.StatusOK, `[{"id":"user-2","email":"bob@example.com","display_name":"bob","avatar_url":"","deleted":false}]`)

	rec = serveRequest(t, Services{}, http.MethodGet, "/me/blocks", "", "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_AddContactBlocked(t *testing.T) {
	uc := NewMockAddContactUseCase(t)
	uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).Return(contact.ErrBlocked).Once()

	rec := serveRequest(t, Services{AddContact: uc}, http.MethodPost, "/me/contacts", `{"contact_id":"user-2"}`, "user-1")
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_UpdateDeletedProfile(t *testing.T) {
	uc := NewMockUpdateProfileUseCase(t)
	uc.EXPECT().Execute(mock.Anything, user.ID("user-1"), mock.Anything).Return(user.User{}, user.ErrDeleted).Once()

	rec := serveRequest(t, Services{UpdateProfile: uc}, http.MethodPatch, "/me", `{"display_name":"x"}`, "user-1")
	require.Equal(t, http.StatusConflict, rec.Code)
}
