package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/contact"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testUser(t *testing.T, id user.ID, email string) user.User {
	t.Helper()

	u, err := user.New(id, email, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return u
}

func assertJSON(t *testing.T, rec *httptest.ResponseRecorder, status int, expected string) {
	t.Helper()

	require.Equal(t, status, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, expected, rec.Body.String())
}

func TestHandler_GetProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		u := testUser(t, "user-1", "alice@example.com")
		getProfile := NewMockGetProfileUseCase(t)
		getProfile.EXPECT().
			Execute(mock.Anything, u.ID()).
			Return(u, nil).
			Once()

		h := NewHandler(getProfile, nil, nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("X-User-ID", u.ID().String())
		rec := httptest.NewRecorder()

		h.GetProfile(rec, req)

		assertJSON(t, rec, http.StatusOK, `{"id":"user-1","email":"alice@example.com","display_name":"alice","avatar_url":""}`)
	})

	t.Run("missing user id", func(t *testing.T) {
		getProfile := NewMockGetProfileUseCase(t)
		h := NewHandler(getProfile, nil, nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rec := httptest.NewRecorder()

		h.GetProfile(rec, req)

		assertJSON(t, rec, http.StatusUnauthorized, `{"error":"missing user id"}`)
	})

	t.Run("not found", func(t *testing.T) {
		getProfile := NewMockGetProfileUseCase(t)
		getProfile.EXPECT().
			Execute(mock.Anything, user.ID("missing")).
			Return(user.User{}, user.ErrNotFound).
			Once()

		h := NewHandler(getProfile, nil, nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("X-User-ID", "missing")
		rec := httptest.NewRecorder()

		h.GetProfile(rec, req)

		assertJSON(t, rec, http.StatusNotFound, `{"error":"user not found"}`)
	})

	t.Run("internal error", func(t *testing.T) {
		getProfile := NewMockGetProfileUseCase(t)
		getProfile.EXPECT().
			Execute(mock.Anything, user.ID("user-1")).
			Return(user.User{}, errors.New("database unavailable")).
			Once()

		h := NewHandler(getProfile, nil, nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.GetProfile(rec, req)

		assertJSON(t, rec, http.StatusInternalServerError, `{"error":"internal error"}`)
	})
}

func TestHandler_UpdateProfile(t *testing.T) {
	t.Run("update both fields", func(t *testing.T) {
		u := testUser(t, "user-1", "alice@example.com")
		now := u.CreatedAt().Add(time.Minute)
		require.NoError(t, u.ChangeDisplayName("Alice Cooper", now))
		require.NoError(t, u.ChangeAvatarURL("https://example.com/avatar.png", now))

		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, u.ID(), mock.MatchedBy(func(input usecase.UpdateProfileInput) bool {
				return input.DisplayName != nil && *input.DisplayName == "Alice Cooper" &&
					input.AvatarURL != nil && *input.AvatarURL == "https://example.com/avatar.png"
			})).
			Return(u, nil).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Alice Cooper","avatar_url":"https://example.com/avatar.png"}`))
		req.Header.Set("X-User-ID", u.ID().String())
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusOK, `{"id":"user-1","email":"alice@example.com","display_name":"Alice Cooper","avatar_url":"https://example.com/avatar.png"}`)
	})

	t.Run("update display name only", func(t *testing.T) {
		u := testUser(t, "user-1", "alice@example.com")
		require.NoError(t, u.ChangeDisplayName("Alice Cooper", u.CreatedAt().Add(time.Minute)))

		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, u.ID(), mock.MatchedBy(func(input usecase.UpdateProfileInput) bool {
				return input.DisplayName != nil && *input.DisplayName == "Alice Cooper" && input.AvatarURL == nil
			})).
			Return(u, nil).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Alice Cooper"}`))
		req.Header.Set("X-User-ID", u.ID().String())
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusOK, `{"id":"user-1","email":"alice@example.com","display_name":"Alice Cooper","avatar_url":""}`)
	})

	t.Run("clear avatar", func(t *testing.T) {
		u := testUser(t, "user-1", "alice@example.com")

		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, u.ID(), mock.MatchedBy(func(input usecase.UpdateProfileInput) bool {
				return input.DisplayName == nil && input.AvatarURL != nil && *input.AvatarURL == ""
			})).
			Return(u, nil).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"avatar_url":""}`))
		req.Header.Set("X-User-ID", u.ID().String())
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusOK, `{"id":"user-1","email":"alice@example.com","display_name":"alice","avatar_url":""}`)
	})

	t.Run("missing user id", func(t *testing.T) {
		updateProfile := NewMockUpdateProfileUseCase(t)
		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Alice"}`))
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusUnauthorized, `{"error":"missing user id"}`)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		updateProfile := NewMockUpdateProfileUseCase(t)
		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusBadRequest, `{"error":"invalid request body"}`)
	})

	t.Run("empty display name", func(t *testing.T) {
		name := "  "
		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), usecase.UpdateProfileInput{DisplayName: &name}).
			Return(user.User{}, user.ErrDisplayNameEmpty).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"  "}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusBadRequest, `{"error":"`+user.ErrDisplayNameEmpty.Error()+`"}`)
	})

	t.Run("not found", func(t *testing.T) {
		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, user.ID("missing"), mock.Anything).
			Return(user.User{}, user.ErrNotFound).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Alice"}`))
		req.Header.Set("X-User-ID", "missing")
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusNotFound, `{"error":"user not found"}`)
	})

	t.Run("internal error", func(t *testing.T) {
		updateProfile := NewMockUpdateProfileUseCase(t)
		updateProfile.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), mock.Anything).
			Return(user.User{}, errors.New("database unavailable")).
			Once()

		h := NewHandler(nil, updateProfile, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Alice"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assertJSON(t, rec, http.StatusInternalServerError, `{"error":"internal error"}`)
	})
}

func TestHandler_SearchUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		alice := testUser(t, "user-1", "alice@example.com")
		bob := testUser(t, "user-2", "bob@example.com")
		searchUsers := NewMockSearchUsersUseCase(t)
		searchUsers.EXPECT().
			Execute(mock.Anything, "alice", 20).
			Return([]user.User{alice, bob}, nil).
			Once()

		h := NewHandler(nil, nil, searchUsers, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/search?q=alice", nil)
		rec := httptest.NewRecorder()

		h.SearchUsers(rec, req)

		assertJSON(t, rec, http.StatusOK, `[
			{"id":"user-1","email":"alice@example.com","display_name":"alice","avatar_url":""},
			{"id":"user-2","email":"bob@example.com","display_name":"bob","avatar_url":""}
		]`)
	})

	t.Run("empty query", func(t *testing.T) {
		searchUsers := NewMockSearchUsersUseCase(t)
		searchUsers.EXPECT().
			Execute(mock.Anything, "", 20).
			Return(nil, nil).
			Once()

		h := NewHandler(nil, nil, searchUsers, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		rec := httptest.NewRecorder()

		h.SearchUsers(rec, req)

		assertJSON(t, rec, http.StatusOK, `[]`)
	})

	t.Run("internal error", func(t *testing.T) {
		searchUsers := NewMockSearchUsersUseCase(t)
		searchUsers.EXPECT().
			Execute(mock.Anything, "alice", 20).
			Return(nil, errors.New("database unavailable")).
			Once()

		h := NewHandler(nil, nil, searchUsers, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/search?q=alice", nil)
		rec := httptest.NewRecorder()

		h.SearchUsers(rec, req)

		assertJSON(t, rec, http.StatusInternalServerError, `{"error":"internal error"}`)
	})
}

func TestHandler_AddContact(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		addContact.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).
			Return(nil).
			Once()

		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"user-2"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusOK, `{"status":"success"}`)
	})

	t.Run("missing user id", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"user-2"}`))
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusUnauthorized, `{"error":"missing user id"}`)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusBadRequest, `{"error":"invalid request body"}`)
	})

	t.Run("missing contact id", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusBadRequest, `{"error":"contact_id is required"}`)
	})

	t.Run("cannot add self", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		addContact.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), user.ID("user-1")).
			Return(contact.ErrCannotAddSelf).
			Once()

		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"user-1"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusBadRequest, `{"error":"`+contact.ErrCannotAddSelf.Error()+`"}`)
	})

	t.Run("already exists", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		addContact.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).
			Return(contact.ErrAlreadyExists).
			Once()

		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"user-2"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusConflict, `{"error":"`+contact.ErrAlreadyExists.Error()+`"}`)
	})

	t.Run("contact user not found", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		addContact.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), user.ID("missing")).
			Return(user.ErrNotFound).
			Once()

		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"missing"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusNotFound, `{"error":"contact user not found"}`)
	})

	t.Run("internal error", func(t *testing.T) {
		addContact := NewMockAddContactUseCase(t)
		addContact.EXPECT().
			Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).
			Return(errors.New("database unavailable")).
			Once()

		h := NewHandler(nil, nil, nil, addContact, nil)
		req := httptest.NewRequest(http.MethodPost, "/me/contacts", strings.NewReader(`{"contact_id":"user-2"}`))
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.AddContact(rec, req)

		assertJSON(t, rec, http.StatusInternalServerError, `{"error":"internal error"}`)
	})
}

func TestHandler_GetContacts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		alice := testUser(t, "user-2", "alice@example.com")
		bob := testUser(t, "user-3", "bob@example.com")
		getContacts := NewMockGetContactsUseCase(t)
		getContacts.EXPECT().
			Execute(mock.Anything, user.ID("user-1")).
			Return([]user.User{alice, bob}, nil).
			Once()

		h := NewHandler(nil, nil, nil, nil, getContacts)
		req := httptest.NewRequest(http.MethodGet, "/me/contacts", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.GetContacts(rec, req)

		assertJSON(t, rec, http.StatusOK, `[
			{"id":"user-2","email":"alice@example.com","display_name":"alice","avatar_url":""},
			{"id":"user-3","email":"bob@example.com","display_name":"bob","avatar_url":""}
		]`)
	})

	t.Run("empty contacts", func(t *testing.T) {
		getContacts := NewMockGetContactsUseCase(t)
		getContacts.EXPECT().
			Execute(mock.Anything, user.ID("user-1")).
			Return(nil, nil).
			Once()

		h := NewHandler(nil, nil, nil, nil, getContacts)
		req := httptest.NewRequest(http.MethodGet, "/me/contacts", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.GetContacts(rec, req)

		assertJSON(t, rec, http.StatusOK, `[]`)
	})

	t.Run("missing user id", func(t *testing.T) {
		getContacts := NewMockGetContactsUseCase(t)
		h := NewHandler(nil, nil, nil, nil, getContacts)
		req := httptest.NewRequest(http.MethodGet, "/me/contacts", nil)
		rec := httptest.NewRecorder()

		h.GetContacts(rec, req)

		assertJSON(t, rec, http.StatusUnauthorized, `{"error":"missing user id"}`)
	})

	t.Run("internal error", func(t *testing.T) {
		getContacts := NewMockGetContactsUseCase(t)
		getContacts.EXPECT().
			Execute(mock.Anything, user.ID("user-1")).
			Return(nil, errors.New("database unavailable")).
			Once()

		h := NewHandler(nil, nil, nil, nil, getContacts)
		req := httptest.NewRequest(http.MethodGet, "/me/contacts", nil)
		req.Header.Set("X-User-ID", "user-1")
		rec := httptest.NewRecorder()

		h.GetContacts(rec, req)

		assertJSON(t, rec, http.StatusInternalServerError, `{"error":"internal error"}`)
	})
}
