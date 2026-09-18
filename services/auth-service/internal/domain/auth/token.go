package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired      = errors.New("token expired")
	ErrTokenInvalid      = errors.New("token invalid")
	ErrTokenMalformed    = errors.New("token malformed")
	ErrRefreshExpired    = errors.New("refresh token expired")
	ErrRefreshInvalid    = errors.New("refresh token invalid")
	ErrPrivateKeyInvalid = errors.New("invalid private key format")
	ErrTokenIssueFailed  = errors.New("failed to issue access token")
)

type (
	AccessToken  string
	RefreshToken string
	Token        string
)

type TokenPair struct {
	Access    AccessToken
	Refresh   RefreshToken
	ExpiresAt time.Time
}

type Identity struct {
	UserID    string
	Email     string
	ExpiresAt time.Time
}

type RefreshEntry struct {
	ID        string
	UserID    UserID
	Token     RefreshToken
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewRefreshEntry(userID UserID, ttl time.Duration) RefreshEntry {
	now := time.Now().UTC()
	return RefreshEntry{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     RefreshToken(uuid.New().String()),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
}

func (e RefreshEntry) IsExpired() bool {
	return time.Now().UTC().After(e.ExpiresAt)
}

type TokenService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(
	privateKeyPEM []byte,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (*TokenService, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrivateKeyInvalid, err)
	}

	return &TokenService{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

type CustomClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *TokenService) Issue(user User) (TokenPair, RefreshEntry, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTTL)

	claims := CustomClaims{
		Email:     user.Email,
		Subject:   string(user.ID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return TokenPair{}, RefreshEntry{}, fmt.Errorf("%w: %v", ErrTokenIssueFailed, err)
	}

	refresh := NewRefreshEntry(user.ID, s.refreshTTL)

	return TokenPair{
		Access:    AccessToken(signed),
		Refresh:   refresh.Token,
		ExpiresAt: expiresAt,
	}, refresh, nil
}

func (s *TokenService) PublicKeyPEM() (*rsa.PublicKey, error) {
	return s.publicKey, nil
}

func (s *TokenService) Authenticate(t Token) (Identity, error) {
	parsed, err := jwt.ParseWithClaims(string(t), &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrTokenInvalid
		}
		return s.publicKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return Identity{}, ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return Identity{}, ErrTokenMalformed
		default:
			return Identity{}, ErrTokenInvalid
		}
	}

	claims, ok := parsed.Claims.(*CustomClaims)
	if !ok || !parsed.Valid {
		return Identity{}, ErrTokenInvalid
	}

	return Identity{
		UserID:    claims.Subject,
		Email:     claims.Email,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
