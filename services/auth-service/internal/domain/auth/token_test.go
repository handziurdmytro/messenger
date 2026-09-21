package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestRSAKeyPEM(t *testing.T) []byte {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
}

func TestRefreshEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"active", time.Now().Add(time.Minute), false},
		{"expired", time.Now().Add(-time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := RefreshEntry{ExpiresAt: tt.expiresAt}

			assert.Equal(t, tt.want, entry.IsExpired())
		})
	}
}

func TestNewRefreshEntry(t *testing.T) {
	userID := UserID("test-user-id")
	ttl := 24 * time.Hour

	before := time.Now()
	entry := NewRefreshEntry(userID, ttl)
	after := time.Now()

	require.NotEmpty(t, entry.ID)
	require.NotEmpty(t, entry.Token)

	assert.Equal(t, userID, entry.UserID)
	assert.WithinRange(t, entry.CreatedAt, before, after)
	assert.Equal(t, entry.CreatedAt.Add(ttl), entry.ExpiresAt)
}

func TestNewTokenService(t *testing.T) {
	validPEM := generateTestRSAKeyPEM(t)

	tests := []struct {
		name    string
		pem     []byte
		wantErr error
	}{
		{"valid key", validPEM, nil},
		{"invalid key", []byte("invalid-pem-data"), ErrPrivateKeyInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewTokenService(
				tt.pem,
				15*time.Minute,
				24*time.Hour,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, svc)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, svc)
		})
	}
}

func TestTokenService_Issue(t *testing.T) {
	const (
		accessTTL  = 15 * time.Minute
		refreshTTL = 24 * time.Hour
	)

	svc, err := NewTokenService(
		generateTestRSAKeyPEM(t),
		accessTTL,
		refreshTTL,
	)
	require.NoError(t, err)

	user := User{
		ID:    UserID("test-id"),
		Email: "test@example.com",
	}

	before := time.Now()
	pair, refresh, err := svc.Issue(user)
	after := time.Now()

	require.NoError(t, err)

	require.NotEmpty(t, pair.Access)
	require.NotEmpty(t, pair.Refresh)

	assert.Equal(t, refresh.Token, pair.Refresh)
	assert.Equal(t, user.ID, refresh.UserID)

	assert.WithinRange(
		t,
		pair.ExpiresAt,
		before.Add(accessTTL),
		after.Add(accessTTL),
	)

	assert.WithinRange(t, refresh.CreatedAt, before, after)
	assert.Equal(t, refresh.CreatedAt.Add(refreshTTL), refresh.ExpiresAt)
}

func TestTokenService_Authenticate(t *testing.T) {
	svc, err := NewTokenService(
		generateTestRSAKeyPEM(t),
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	user := User{
		ID:    UserID("test-id"),
		Email: "test@example.com",
	}

	pair, _, err := svc.Issue(user)
	require.NoError(t, err)

	expiredJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, CustomClaims{
		Email:     user.Email,
		Subject:   string(user.ID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	})

	expiredToken, err := expiredJWT.SignedString(svc.privateKey)
	require.NoError(t, err)

	otherPEM := generateTestRSAKeyPEM(t)
	otherSvc, err := NewTokenService(otherPEM, time.Minute, time.Hour)
	require.NoError(t, err)

	invalidJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, CustomClaims{
		Email:     user.Email,
		Subject:   string(user.ID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	invalidToken, err := invalidJWT.SignedString(otherSvc.privateKey)
	require.NoError(t, err)

	wrongAlgorithmJWT := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": string(user.ID),
			"exp": time.Now().Add(time.Hour).Unix(),
		},
	)

	wrongAlgorithmToken, err := wrongAlgorithmJWT.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   Token
		wantErr error
	}{
		{"success", Token(pair.Access), nil},
		{"malformed", Token("invalid token"), ErrTokenMalformed},
		{"expired", Token(expiredToken), ErrTokenExpired},
		{"invalid signature", Token(invalidToken), ErrTokenInvalid},
		{"wrong algorithm", Token(wrongAlgorithmToken), ErrTokenInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := svc.Authenticate(tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, Identity{}, identity)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, string(user.ID), identity.UserID)
			assert.Equal(t, user.Email, identity.Email)
			assert.Equal(t, pair.ExpiresAt.Unix(), identity.ExpiresAt.Unix())
		})
	}
}

func TestTokenService_PublicKeyPEM(t *testing.T) {
	svc, err := NewTokenService(
		generateTestRSAKeyPEM(t),
		15*time.Minute,
		24*time.Hour,
	)
	require.NoError(t, err)

	publicKey, err := svc.PublicKeyPEM()

	require.NoError(t, err)
	require.NotNil(t, publicKey)

	assert.Equal(t, svc.privateKey.PublicKey.N, publicKey.N)
	assert.Equal(t, svc.privateKey.PublicKey.E, publicKey.E)
}
