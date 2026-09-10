package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"
)

func generateTestRSAKeyPEM(t *testing.T) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() setup error = %v, wantErr nil", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
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
			e := RefreshEntry{ExpiresAt: tt.expiresAt}
			if got := e.IsExpired(); got != tt.want {
				t.Errorf("RefreshEntry.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRefreshEntry(t *testing.T) {
	userID := UserID("test-user-id")
	ttl := 24 * time.Hour

	entry := NewRefreshEntry(userID, ttl)

	if entry.ID == "" {
		t.Errorf("NewRefreshEntry().ID = %v, want non-empty", entry.ID)
	}
	if entry.UserID != userID {
		t.Errorf("NewRefreshEntry().UserID = %v, want %v", entry.UserID, userID)
	}
	if entry.Token == "" {
		t.Errorf("NewRefreshEntry().Token = %v, want non-empty", entry.Token)
	}

	expectedExpires := time.Now().UTC().Add(ttl)
	if diff := entry.ExpiresAt.Sub(expectedExpires); diff > time.Second || diff < -time.Second {
		t.Errorf("NewRefreshEntry().ExpiresAt = %v, want %v", entry.ExpiresAt, expectedExpires)
	}
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
			_, err := NewTokenService(tt.pem, time.Minute, time.Hour)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewTokenService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenService_Issue(t *testing.T) {
	validPEM := generateTestRSAKeyPEM(t)
	svc, err := NewTokenService(validPEM, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() setup error = %v, wantErr nil", err)
	}

	user := User{
		ID:    UserID("test-id"),
		Email: "test@example.com",
	}

	pair, refresh, err := svc.Issue(user)

	if err != nil {
		t.Fatalf("TokenService.Issue() error = %v, wantErr nil", err)
	}
	if pair.Access == "" {
		t.Errorf("TokenService.Issue().Access = %v, want non-empty", pair.Access)
	}
	if pair.Refresh == "" {
		t.Errorf("TokenService.Issue().Refresh = %v, want non-empty", pair.Refresh)
	}
	if refresh.Token != pair.Refresh {
		t.Errorf("TokenService.Issue() refresh token mismatch: entry = %v, pair = %v", refresh.Token, pair.Refresh)
	}
	if refresh.UserID != user.ID {
		t.Errorf("TokenService.Issue().RefreshEntry.UserID = %v, want %v", refresh.UserID, user.ID)
	}
}
