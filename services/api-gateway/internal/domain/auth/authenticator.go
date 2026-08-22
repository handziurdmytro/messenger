package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator struct {
	publicKey *rsa.PublicKey
	clock     func() time.Time
}

func NewAuthenticator(publicKey []byte) (*Authenticator, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	return &Authenticator{
		publicKey: key,
		clock:     time.Now,
	}, nil
}

func (a *Authenticator) Authenticate(t Token) (Identity, error) {
	claims, err := a.parse(t)
	if err != nil {
		return Identity{}, fmt.Errorf("failed to authenticate: %w", err)
	}

	if err := a.validate(claims); err != nil {
		return Identity{}, fmt.Errorf("failed to authenticate: %w", err)
	}

	return claims.ToIdentity(), nil
}

func (a *Authenticator) parse(token Token) (Claims, error) {
	parsed, err := jwt.Parse(string(token), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrTokenInvalid
		}
		return a.publicKey, nil
	})
	if err != nil {
		return Claims{}, a.mapError(err)
	}

	raw, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrTokenMalformed
	}

	return a.extractClaims(raw)
}

func (a *Authenticator) extractClaims(raw jwt.MapClaims) (Claims, error) {
	userID, ok := raw["sub"].(string)
	if !ok {
		return Claims{}, ErrTokenMalformed
	}

	email, _ := raw["email"].(string)

	exp, ok := raw["exp"].(float64)
	if !ok {
		return Claims{}, ErrTokenMalformed
	}

	iat, ok := raw["iat"].(float64)
	if !ok {
		return Claims{}, ErrTokenMalformed
	}

	roles, err := a.extractRoles(raw)
	if err != nil {
		return Claims{}, fmt.Errorf("failed to extract claims: %w", err)
	}

	return Claims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		ExpiresAt: time.Unix(int64(exp), 0),
		IssuedAt:  time.Unix(int64(iat), 0),
	}, nil
}

func (a *Authenticator) extractRoles(raw jwt.MapClaims) ([]Role, error) {
	rawRoles, ok := raw["roles"].([]any)
	if !ok {
		return []Role{RoleUser}, nil
	}

	roles := make([]Role, 0, len(rawRoles))
	for _, r := range rawRoles {
		roleStr, ok := r.(string)
		if !ok {
			return nil, ErrTokenMalformed
		}
		roles = append(roles, Role(roleStr))
	}
	return roles, nil
}

func (a *Authenticator) validate(c Claims) error {
	if a.clock().After(c.ExpiresAt) {
		return ErrTokenExpired
	}
	return nil
}

func (a *Authenticator) mapError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenMalformed):
		return ErrTokenMalformed
	default:
		return ErrTokenInvalid
	}
}
