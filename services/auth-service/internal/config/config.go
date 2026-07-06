package config

import (
	"fmt"
	"strconv"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	PostgresDSN      string
	PrivateKeyPath   string
	KafkaBrokers     string
	KafkaTopic       string
	AccessTTLMinutes int
	RefreshTTLDays   int
	LogLevel         string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	serverPort, err := env.Get("SERVER_PORT")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load SERVER_PORT: %w", err)
	}

	postgresDSN, err := env.Get("POSTGRES_DSN")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load POSTGRES_DSN: %w", err)
	}

	privateKeyPath, err := env.Get("PRIVATE_KEY_PATH")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load PRIVATE_KEY_PATH: %w", err)
	}

	kafkaBrokers, err := env.Get("KAFKA_BROKERS")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load KAFKA_BROKERS: %w", err)
	}

	kafkaTopic, err := env.Get("KAFKA_TOPIC")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load KAFKA_TOPIC: %w", err)
	}

	accessTTLStr := env.GetOr("ACCESS_TTL_MINUTES", "15")
	accessTTL, err := strconv.Atoi(accessTTLStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid ACCESS_TTL_MINUTES format: %w", err)
	}

	refreshTTLStr := env.GetOr("REFRESH_TTL_DAYS", "30")
	refreshTTL, err := strconv.Atoi(refreshTTLStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid REFRESH_TTL_DAYS format: %w", err)
	}

	return Config{
		ServerPort:       serverPort,
		PostgresDSN:      postgresDSN,
		PrivateKeyPath:   privateKeyPath,
		KafkaBrokers:     kafkaBrokers,
		KafkaTopic:       kafkaTopic,
		AccessTTLMinutes: accessTTL,
		RefreshTTLDays:   refreshTTL,
		LogLevel:         env.GetOr("LOG_LEVEL", "INFO"),
	}, nil
}
