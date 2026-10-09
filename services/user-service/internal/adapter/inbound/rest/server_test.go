package rest

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServer_Routes(t *testing.T) {
	profile, err := user.New("user-1", "user@example.com", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		setupMocks func(*MockGetProfileUseCase, *MockUpdateProfileUseCase, *MockSearchUsersUseCase, *MockAddContactUseCase, *MockGetContactsUseCase)
	}{
		{
			name:   "get profile",
			method: http.MethodGet,
			path:   "/me",
			setupMocks: func(getProfile *MockGetProfileUseCase, _ *MockUpdateProfileUseCase, _ *MockSearchUsersUseCase, _ *MockAddContactUseCase, _ *MockGetContactsUseCase) {
				getProfile.EXPECT().
					Execute(mock.Anything, user.ID("user-1")).
					Return(profile, nil).
					Once()
			},
		},
		{
			name:   "update profile",
			method: http.MethodPatch,
			path:   "/me",
			body:   `{}`,
			setupMocks: func(_ *MockGetProfileUseCase, updateProfile *MockUpdateProfileUseCase, _ *MockSearchUsersUseCase, _ *MockAddContactUseCase, _ *MockGetContactsUseCase) {
				updateProfile.EXPECT().
					Execute(mock.Anything, user.ID("user-1"), usecase.UpdateProfileInput{}).
					Return(profile, nil).
					Once()
			},
		},
		{
			name:   "search users",
			method: http.MethodGet,
			path:   "/search?q=alice",
			setupMocks: func(_ *MockGetProfileUseCase, _ *MockUpdateProfileUseCase, searchUsers *MockSearchUsersUseCase, _ *MockAddContactUseCase, _ *MockGetContactsUseCase) {
				searchUsers.EXPECT().
					Execute(mock.Anything, "alice", 20).
					Return([]user.User{}, nil).
					Once()
			},
		},
		{
			name:   "add contact",
			method: http.MethodPost,
			path:   "/me/contacts",
			body:   `{"contact_id":"user-2"}`,
			setupMocks: func(_ *MockGetProfileUseCase, _ *MockUpdateProfileUseCase, _ *MockSearchUsersUseCase, addContact *MockAddContactUseCase, _ *MockGetContactsUseCase) {
				addContact.EXPECT().
					Execute(mock.Anything, user.ID("user-1"), user.ID("user-2")).
					Return(nil).
					Once()
			},
		},
		{
			name:   "get contacts",
			method: http.MethodGet,
			path:   "/me/contacts",
			setupMocks: func(_ *MockGetProfileUseCase, _ *MockUpdateProfileUseCase, _ *MockSearchUsersUseCase, _ *MockAddContactUseCase, getContacts *MockGetContactsUseCase) {
				getContacts.EXPECT().
					Execute(mock.Anything, user.ID("user-1")).
					Return([]user.User{}, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getProfile := NewMockGetProfileUseCase(t)
			updateProfile := NewMockUpdateProfileUseCase(t)
			searchUsers := NewMockSearchUsersUseCase(t)
			addContact := NewMockAddContactUseCase(t)
			getContacts := NewMockGetContactsUseCase(t)

			tt.setupMocks(getProfile, updateProfile, searchUsers, addContact, getContacts)

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			server := NewServer("8080", getProfile, updateProfile, searchUsers, addContact, getContacts, log)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("X-User-ID", "user-1")
			rec := httptest.NewRecorder()

			server.http.Handler.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestServer_UnknownRoute(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer("8080", nil, nil, nil, nil, nil, log)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	server.http.Handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_MethodNotAllowed(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer("8080", nil, nil, nil, nil, nil, log)

	req := httptest.NewRequest(http.MethodDelete, "/me", nil)
	rec := httptest.NewRecorder()

	server.http.Handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Contains(t, rec.Header().Get("Allow"), http.MethodGet)
	require.Contains(t, rec.Header().Get("Allow"), http.MethodPatch)
}
