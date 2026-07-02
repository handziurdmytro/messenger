package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordInvalid  = errors.New("invalid password")
)

type PasswordHash string

func HashPassword(plain string) (PasswordHash, error) {
	if len(plain) < 8 {
		return "", ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return PasswordHash(hash), nil
}

func CheckPassword(plain string, hash PasswordHash) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordInvalid
	}
	return err
}
