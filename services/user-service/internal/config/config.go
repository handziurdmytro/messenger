package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort   string
	PostgresDSN  string
	KafkaBrokers string
	KafkaTopic   string
	KafkaGroupID string
	LogLevel     string
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

	kafkaBrokers, err := env.Get("KAFKA_BROKERS")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load KAFKA_BROKERS: %w", err)
	}

	kafkaTopic, err := env.Get("KAFKA_TOPIC")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load KAFKA_TOPIC: %w", err)
	}

	kafkaGroupID, err := env.Get("KAFKA_GROUP_ID")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load KAFKA_GROUP_ID: %w", err)
	}

	return Config{
		ServerPort:   serverPort,
		PostgresDSN:  postgresDSN,
		KafkaBrokers: kafkaBrokers,
		KafkaTopic:   kafkaTopic,
		KafkaGroupID: kafkaGroupID,
		LogLevel:     env.GetOr("LOG_LEVEL", "INFO"),
	}, nil
}
