package rest

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServerIntegration_Routing(t *testing.T) {
	routeErr := errors.New("route reached")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		setupMocks func(*MockRegisterService, *MockLoginService, *MockRefreshService, *MockTokenService)
		wantStatus int
		wantAllow  string
	}{
		{
			name:   "register",
			method: http.MethodPost,
			path:   "/register",
			body:   `{"email":"test@example.com","password":"password"}`,
			setupMocks: func(register *MockRegisterService, _ *MockLoginService, _ *MockRefreshService, _ *MockTokenService) {
				register.EXPECT().
					Execute(mock.Anything, usecase.RegisterInput{
						Email:    "test@example.com",
						Password: "password",
					}).
					Return(usecase.RegisterOutput{}, routeErr).
					Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "login",
			method: http.MethodPost,
			path:   "/login",
			body:   `{"email":"test@example.com","password":"password"}`,
			setupMocks: func(_ *MockRegisterService, login *MockLoginService, _ *MockRefreshService, _ *MockTokenService) {
				login.EXPECT().
					Execute(mock.Anything, usecase.LoginInput{
						Email:    "test@example.com",
						Password: "password",
					}).
					Return(usecase.LoginOutput{}, routeErr).
					Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "refresh",
			method: http.MethodPost,
			path:   "/refresh",
			body:   `{"refresh_token":"refresh123"}`,
			setupMocks: func(_ *MockRegisterService, _ *MockLoginService, refresh *MockRefreshService, _ *MockTokenService) {
				refresh.EXPECT().
					Execute(mock.Anything, auth.RefreshToken("refresh123")).
					Return(auth.TokenPair{}, routeErr).
					Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "validate",
			method: http.MethodGet,
			path:   "/validate?token=valid123",
			setupMocks: func(_ *MockRegisterService, _ *MockLoginService, _ *MockRefreshService, token *MockTokenService) {
				token.EXPECT().
					Authenticate(auth.Token("valid123")).
					Return(auth.Identity{UserID: "user123"}, nil).
					Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "jwks",
			method: http.MethodGet,
			path:   "/.well-known/jwks.pem",
			setupMocks: func(_ *MockRegisterService, _ *MockLoginService, _ *MockRefreshService, token *MockTokenService) {
				token.EXPECT().PublicKeyPEM().Return(nil, routeErr).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "register wrong method",
			method:     http.MethodGet,
			path:       "/register",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name:       "login wrong method",
			method:     http.MethodGet,
			path:       "/login",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name:       "refresh wrong method",
			method:     http.MethodGet,
			path:       "/refresh",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
		{
			name:       "validate wrong method",
			method:     http.MethodPost,
			path:       "/validate",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet,
		},
		{
			name:       "jwks wrong method",
			method:     http.MethodPost,
			path:       "/.well-known/jwks.pem",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			register := NewMockRegisterService(t)
			login := NewMockLoginService(t)
			refresh := NewMockRefreshService(t)
			token := NewMockTokenService(t)

			if tt.setupMocks != nil {
				tt.setupMocks(register, login, refresh, token)
			}

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			app := NewServer("0", register, login, refresh, token, log)

			server := httptest.NewServer(app.http.Handler)
			t.Cleanup(server.Close)

			req, err := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
			require.NoError(t, err)

			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := server.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantAllow != "" {
				allow := strings.Split(resp.Header.Get("Allow"), ",")
				for i := range allow {
					allow[i] = strings.TrimSpace(allow[i])
				}
				require.Contains(t, allow, tt.wantAllow)
			}

			if tt.name == "validate" {
				require.Equal(t, "user123", resp.Header.Get("X-User-ID"))
			}
		})
	}
}

func TestServerIntegration_Swagger(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := NewServer("0", nil, nil, nil, nil, log)

	server := httptest.NewServer(app.http.Handler)
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/swagger/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}
