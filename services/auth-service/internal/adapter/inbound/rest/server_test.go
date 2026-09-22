package rest

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testServerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewServer(t *testing.T) {
	server := NewServer("80", nil, nil, nil, nil, testServerLogger())

	require.NotNil(t, server)
	require.NotNil(t, server.http)
	require.NotNil(t, server.http.Handler)
	require.Equal(t, ":80", server.http.Addr)
	require.Equal(t, 10*time.Second, server.http.ReadTimeout)
	require.Equal(t, 10*time.Second, server.http.WriteTimeout)
	require.Equal(t, 60*time.Second, server.http.IdleTimeout)
}

func TestServer_Routes(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		setupRegister func(*MockRegisterService)
		setupLogin    func(*MockLoginService)
		setupRefresh  func(*MockRefreshService)
		setupToken    func(*MockTokenService)
		wantStatus    int
		wantUserID    string
	}{
		{
			name:   "validate",
			method: http.MethodGet,
			path:   "/validate?token=valid123",
			setupToken: func(s *MockTokenService) {
				s.EXPECT().Authenticate(auth.Token("valid123")).
					Return(auth.Identity{UserID: "user123"}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantUserID: "user123",
		},
		{
			name:   "register",
			method: http.MethodPost,
			path:   "/register",
			body:   `{"email":"test@example.com","password":"password"}`,
			setupRegister: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.RegisterOutput{}, nil).Once()
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:   "login",
			method: http.MethodPost,
			path:   "/login",
			body:   `{"email":"test@example.com","password":"password"}`,
			setupLogin: func(s *MockLoginService) {
				s.EXPECT().Execute(mock.Anything, usecase.LoginInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.LoginOutput{}, nil).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "refresh",
			method: http.MethodPost,
			path:   "/refresh",
			body:   `{"refresh_token":"refresh123"}`,
			setupRefresh: func(s *MockRefreshService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh123")).
					Return(auth.TokenPair{}, nil).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "jwks",
			method: http.MethodGet,
			path:   "/.well-known/jwks.pem",
			setupToken: func(s *MockTokenService) {
				s.EXPECT().PublicKeyPEM().Return(nil, errors.New("key unavailable")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			register := NewMockRegisterService(t)
			login := NewMockLoginService(t)
			refresh := NewMockRefreshService(t)
			token := NewMockTokenService(t)

			if tt.setupRegister != nil {
				tt.setupRegister(register)
			}
			if tt.setupLogin != nil {
				tt.setupLogin(login)
			}
			if tt.setupRefresh != nil {
				tt.setupRefresh(refresh)
			}
			if tt.setupToken != nil {
				tt.setupToken(token)
			}

			server := NewServer("0", register, login, refresh, token, testServerLogger())
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()

			server.http.Handler.ServeHTTP(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()
			require.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantUserID != "" {
				require.Equal(t, tt.wantUserID, resp.Header.Get("X-User-ID"))
			}
		})
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{"register", http.MethodGet, "/register", http.MethodPost},
		{"login", http.MethodGet, "/login", http.MethodPost},
		{"refresh", http.MethodGet, "/refresh", http.MethodPost},
		{"validate", http.MethodPost, "/validate", http.MethodGet + ", " + http.MethodHead},
		{"jwks", http.MethodPost, "/.well-known/jwks.pem", http.MethodGet + ", " + http.MethodHead},
	}

	server := NewServer("0", nil, nil, nil, nil, testServerLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			server.http.Handler.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
			require.Equal(t, tt.wantAllow, recorder.Header().Get("Allow"))
		})
	}
}

func TestServer_NotFound(t *testing.T) {
	server := NewServer("0", nil, nil, nil, nil, testServerLogger())
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	recorder := httptest.NewRecorder()

	server.http.Handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestServer_SwaggerRoute(t *testing.T) {
	server := NewServer("0", nil, nil, nil, nil, testServerLogger())
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	recorder := httptest.NewRecorder()

	server.http.Handler.ServeHTTP(recorder, req)

	require.NotEqual(t, http.StatusNotFound, recorder.Code)
	require.NotEqual(t, http.StatusMethodNotAllowed, recorder.Code)
}
