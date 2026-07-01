package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	WSPort        string
	RESTPort      string
	LogLevel      string
	KafkaBrokers  string
	KafkaInTopic  string
	KafkaOutTopic string
	KafkaGroupID  string
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

	kafkaInTopic, err := env.Get("KAFKA_IN_TOPIC")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	kafkaOutTopic, err := env.Get("KAFKA_OUT_TOPIC")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	kafkaGroupID, err := env.Get("KAFKA_GROUP_ID")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	return Config{
		WSPort:        wsPort,
		RESTPort:      restPort,
		LogLevel:      env.GetOr("LOG_LEVEL", "INFO"),
		KafkaBrokers:  kafkaBrokers,
		KafkaInTopic:  kafkaInTopic,
		KafkaOutTopic: kafkaOutTopic,
		KafkaGroupID:  kafkaGroupID,
	}, nil
}
