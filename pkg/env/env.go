package env

import (
	"errors"
	"fmt"
	"os"
)

var ErrRequired = errors.New("environment variable is required")

func Get(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s, %w", key, ErrRequired)
	}
	return v, nil
}

func GetOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
