package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	WSPort       string
	RESTPort     string
	LogLevel     string
	KafkaBrokers string
	KafkaTopic   string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	wsPort, err := env.Get("WS_PORT")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	restPort, err := env.Get("REST_PORT")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	kafkaBrokers, err := env.Get("KAFKA_BROKERS")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	kafkaTopic, err := env.Get("KAFKA_TOPIC")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	return Config{
		WSPort:       wsPort,
		RESTPort:     restPort,
		LogLevel:     env.GetOr("LOG_LEVEL", "INFO"),
		KafkaBrokers: kafkaBrokers,
		KafkaTopic:   kafkaTopic,
	}, nil
}
