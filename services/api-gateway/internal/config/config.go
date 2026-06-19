package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel      string
	ServerPort    string
	PublicKeyPath string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	serverPort, err := env.Get("SERVER_PORT")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env variables: %w", err)
	}

	pubKeyPath, err := env.Get("PUBLIC_KEY_PATH")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env variables: %w", err)
	}

	return Config{
		LogLevel:      env.GetOr("LOG_LEVEL", "INFO"),
		ServerPort:    serverPort,
		PublicKeyPath: pubKeyPath,
	}, nil
}
