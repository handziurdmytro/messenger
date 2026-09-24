package rest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
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

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		target string
		want   string
	}{
		{"valid", "Bearer token123", "/validate", "token123"},
		{"valid from URL", "", "/validate?token=token123", "token123"},
		{"valid header first", "Bearer token123", "/validate?token=token321", "token123"},
		{"empty", "", "/validate", ""},
		{"invalid", "token123", "/validate", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			require.Equal(t, tt.want, extractToken(req))
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		authorization string
		setupMock     func(*MockTokenService)
		wantStatus    int
		wantUserID    string
	}{
		{
			name:          "success with authorization header",
			url:           "/validate",
			authorization: "Bearer valid123",
			setupMock: func(svc *MockTokenService) {
				svc.EXPECT().Authenticate(auth.Token("valid123")).Return(auth.Identity{UserID: "test-user"}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantUserID: "test-user",
		},
		{
			name: "success with query parameter",
			url:  "/validate?token=valid123",
			setupMock: func(svc *MockTokenService) {
				svc.EXPECT().Authenticate(auth.Token("valid123")).Return(auth.Identity{UserID: "test-user"}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantUserID: "test-user",
		},
		{
			name:       "missing token",
			url:        "/validate",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "invalid token",
			url:           "/validate",
			authorization: "Bearer invalid123",
			setupMock: func(svc *MockTokenService) {
				svc.EXPECT().Authenticate(auth.Token("invalid123")).Return(auth.Identity{}, auth.ErrTokenInvalid).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}

			svc := NewMockTokenService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			recorder := httptest.NewRecorder()
			NewHandler(nil, nil, nil, svc).Validate(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode, "unexpected HTTP status")
			require.Equal(t, tt.wantUserID, resp.Header.Get("X-User-ID"), "unexpected user ID")
		})
	}
}

func TestHandler_Register(t *testing.T) {
	expiresAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	credentials := `{"email":"test@example.com","password":"password"}`

	tests := []struct {
		name       string
		body       string
		setupMock  func(*MockRegisterService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: credentials,
			setupMock: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.RegisterOutput{
					UserID: "uuid1234",
					Tokens: auth.TokenPair{
						Access: "access123", Refresh: "refresh123", ExpiresAt: expiresAt,
					},
				}, nil).Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   `{"access_token":"access123","refresh_token":"refresh123","expires_at":"2026-01-01T12:00:00Z"}`,
		},
		{
			name:       "invalid JSON",
			body:       `{"email":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "email already taken",
			body: credentials,
			setupMock: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.RegisterOutput{}, auth.ErrEmailTaken).Once()
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "password too short",
			body: `{"email":"test@example.com","password":"123"}`,
			setupMock: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "123",
				}).Return(usecase.RegisterOutput{}, auth.ErrPasswordTooShort).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "password too long",
			body: `{"email":"test@example.com","password":"very-long-password"}`,
			setupMock: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "very-long-password",
				}).Return(usecase.RegisterOutput{}, auth.ErrPasswordTooLong).Once()
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal server error",
			body: credentials,
			setupMock: func(s *MockRegisterService) {
				s.EXPECT().Execute(mock.Anything, usecase.RegisterInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.RegisterOutput{}, errors.New("database unavailable")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			svc := NewMockRegisterService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			recorder := httptest.NewRecorder()
			NewHandler(svc, nil, nil, nil).Register(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode, "unexpected HTTP status")
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.wantBody, string(body), "unexpected response body")
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	expiresAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	credentials := `{"email":"test@example.com","password":"password"}`

	tests := []struct {
		name       string
		body       string
		setupMock  func(*MockLoginService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: credentials,
			setupMock: func(s *MockLoginService) {
				s.EXPECT().Execute(mock.Anything, usecase.LoginInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.LoginOutput{
					UserID: "uuid1234",
					Tokens: auth.TokenPair{
						Access: "access123", Refresh: "refresh123", ExpiresAt: expiresAt,
					},
				}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"access_token":"access123","refresh_token":"refresh123","expires_at":"2026-01-01T12:00:00Z"}`,
		},
		{name: "invalid JSON", body: `{"email":`, wantStatus: http.StatusBadRequest},
		{
			name: "user not found",
			body: credentials,
			setupMock: func(s *MockLoginService) {
				s.EXPECT().Execute(mock.Anything, usecase.LoginInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.LoginOutput{}, auth.ErrUserNotFound).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid password",
			body: credentials,
			setupMock: func(s *MockLoginService) {
				s.EXPECT().Execute(mock.Anything, usecase.LoginInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.LoginOutput{}, auth.ErrPasswordInvalid).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "internal server error",
			body: credentials,
			setupMock: func(s *MockLoginService) {
				s.EXPECT().Execute(mock.Anything, usecase.LoginInput{
					Email: "test@example.com", Password: "password",
				}).Return(usecase.LoginOutput{}, errors.New("database unavailable")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			svc := NewMockLoginService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			recorder := httptest.NewRecorder()
			NewHandler(nil, svc, nil, nil).Login(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.wantBody, string(body))
			}
		})
	}
}

func TestHandler_Refresh(t *testing.T) {
	expiresAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	validBody := `{"refresh_token":"refresh123"}`

	tests := []struct {
		name       string
		body       string
		setupMock  func(*MockRefreshService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: validBody,
			setupMock: func(s *MockRefreshService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh123")).Return(auth.TokenPair{
					Access: "new-access", Refresh: "new-refresh", ExpiresAt: expiresAt,
				}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_at":"2026-01-01T12:00:00Z"}`,
		},
		{name: "invalid JSON", body: `{"refresh_token":`, wantStatus: http.StatusBadRequest},
		{
			name: "invalid refresh token",
			body: validBody,
			setupMock: func(s *MockRefreshService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh123")).Return(auth.TokenPair{}, auth.ErrRefreshInvalid).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired refresh token",
			body: validBody,
			setupMock: func(s *MockRefreshService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh123")).Return(auth.TokenPair{}, auth.ErrRefreshExpired).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "internal server error",
			body: validBody,
			setupMock: func(s *MockRefreshService) {
				s.EXPECT().Execute(mock.Anything, auth.RefreshToken("refresh123")).Return(auth.TokenPair{}, errors.New("database unavailable")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			svc := NewMockRefreshService(t)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			recorder := httptest.NewRecorder()
			NewHandler(nil, nil, svc, nil).Refresh(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.wantBody, string(body))
			}
		})
	}
}

func TestHandler_JWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	tests := []struct {
		name       string
		setupMock  func(*MockTokenService)
		wantStatus int
		wantKey    *rsa.PublicKey
	}{
		{
			name: "success",
			setupMock: func(s *MockTokenService) {
				s.EXPECT().PublicKeyPEM().Return(&key.PublicKey, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantKey:    &key.PublicKey,
		},
		{
			name: "public key service error",
			setupMock: func(s *MockTokenService) {
				s.EXPECT().PublicKeyPEM().Return(nil, errors.New("key unavailable")).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "invalid public key",
			setupMock: func(s *MockTokenService) {
				s.EXPECT().PublicKeyPEM().Return(&rsa.PublicKey{}, nil).Once()
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockTokenService(t)
			tt.setupMock(svc)

			req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.pem", nil)
			recorder := httptest.NewRecorder()
			NewHandler(nil, nil, nil, svc).JWKS(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantKey != nil {
				require.Equal(t, "application/x-pem-file", resp.Header.Get("Content-Type"))
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				block, rest := pem.Decode(body)
				require.NotNil(t, block)
				require.Empty(t, rest)
				require.Equal(t, "PUBLIC KEY", block.Type)

				parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
				require.NoError(t, err)
				require.Equal(t, tt.wantKey, parsed)
			}
		})
	}
}
