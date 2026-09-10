package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password exceeds maximum length of 72 bytes")
	ErrPasswordInvalid    = errors.New("invalid password")
	ErrPasswordHashFailed = errors.New("failed to hash password")
)

type PasswordHash string

func HashPassword(plain string) (PasswordHash, error) {
	passBytes := []byte(plain)

	if len(passBytes) < 8 {
		return "", ErrPasswordTooShort
	}

	if len(passBytes) > 72 {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword(passBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", ErrPasswordHashFailed
	}

	return PasswordHash(hash), nil
}

func CheckPassword(plain string, hash PasswordHash) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err != nil {
		return ErrPasswordInvalid
	}
	return nil
}
