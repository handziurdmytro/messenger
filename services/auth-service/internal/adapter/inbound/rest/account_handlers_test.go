package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
)

const (
	testUserID      auth.UserID = "user-1"
	testAccessToken             = "valid-token"
)

var errTestInternal = errors.New("database unavailable")

func authenticatedTokenService(t *testing.T) *MockTokenService {
	t.Helper()

	token := NewMockTokenService(t)
	token.EXPECT().
		Authenticate(auth.Token(testAccessToken)).
		Return(auth.Identity{UserID: string(testUserID)}, nil).
		Maybe()
	return token
}

func serve(t *testing.T, services Services, method, path, body, bearer string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	recorder := httptest.NewRecorder()
	NewServer("0", services, testServerLogger()).http.Handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestRequireAuth(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		resp := serve(t, Services{}, http.MethodPost, "/logout-all", "", "")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("token in query is ignored", func(t *testing.T) {
		resp := serve(t, Services{}, http.MethodPost, "/logout-all?token="+testAccessToken, "", "")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid token", func(t *testing.T) {
		token := NewMockTokenService(t)
		token.EXPECT().Authenticate(auth.Token("bad")).Return(auth.Identity{}, auth.ErrTokenInvalid).Once()

		resp := serve(t, Services{Token: token}, http.MethodPost, "/logout-all", "", "bad")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*MockLogoutService)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"refresh_token":"refresh-1"}`,
			setup: func(s *MockLogoutService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh-1")).Return(nil).Once()
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "empty token",
			body: `{"refresh_token":""}`,
			setup: func(s *MockLogoutService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("")).Return(auth.ErrRefreshInvalid).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: `{"refresh_token":"refresh-1"}`,
			setup: func(s *MockLogoutService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh-1")).Return(errTestInternal).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockLogoutService(t)
			if tt.setup != nil {
				tt.setup(svc)
			}

			resp := serve(t, Services{Logout: svc}, http.MethodPost, "/logout", tt.body, "")
			require.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_LogoutAll(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"success", nil, http.StatusNoContent},
		{"internal error", errTestInternal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockLogoutAllService(t)
			svc.EXPECT().Execute(mock.Anything, testUserID).Return(tt.err).Once()

			services := Services{LogoutAll: svc, Token: authenticatedTokenService(t)}
			resp := serve(t, services, http.MethodPost, "/logout-all", "", testAccessToken)
			require.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_ChangePassword(t *testing.T) {
	const body = `{"current_password":"old-password","new_password":"new-password"}`
	input := usecase.ChangePasswordInput{UserID: testUserID, CurrentPassword: "old-password", NewPassword: "new-password"}

	tests := []struct {
		name       string
		body       string
		err        error
		skipCall   bool
		wantStatus int
	}{
		{name: "success", body: body, wantStatus: http.StatusNoContent},
		{name: "wrong current password", body: body, err: auth.ErrPasswordInvalid, wantStatus: http.StatusForbidden},
		{name: "new password too short", body: body, err: auth.ErrPasswordTooShort, wantStatus: http.StatusBadRequest},
		{name: "new password too long", body: body, err: auth.ErrPasswordTooLong, wantStatus: http.StatusBadRequest},
		{name: "password unchanged", body: body, err: auth.ErrPasswordUnchanged, wantStatus: http.StatusBadRequest},
		{name: "user not found", body: body, err: auth.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", body: body, err: errTestInternal, wantStatus: http.StatusInternalServerError},
		{name: "invalid body", body: `{`, skipCall: true, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockChangePasswordService(t)
			if !tt.skipCall {
				svc.EXPECT().Execute(mock.Anything, input).Return(tt.err).Once()
			}

			services := Services{ChangePassword: svc, Token: authenticatedTokenService(t)}
			resp := serve(t, services, http.MethodPatch, "/password", tt.body, testAccessToken)
			require.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_DeleteAccount(t *testing.T) {
	const body = `{"password":"password1"}`
	input := usecase.DeleteAccountInput{UserID: testUserID, Password: "password1"}

	tests := []struct {
		name       string
		body       string
		err        error
		skipCall   bool
		wantStatus int
	}{
		{name: "success", body: body, wantStatus: http.StatusNoContent},
		{name: "wrong password", body: body, err: auth.ErrPasswordInvalid, wantStatus: http.StatusForbidden},
		{name: "user not found", body: body, err: auth.ErrUserNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", body: body, err: errTestInternal, wantStatus: http.StatusInternalServerError},
		{name: "invalid body", body: `{`, skipCall: true, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockDeleteAccountService(t)
			if !tt.skipCall {
				svc.EXPECT().Execute(mock.Anything, input).Return(tt.err).Once()
			}

			services := Services{DeleteAccount: svc, Token: authenticatedTokenService(t)}
			resp := serve(t, services, http.MethodDelete, "/account", tt.body, testAccessToken)
			require.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
