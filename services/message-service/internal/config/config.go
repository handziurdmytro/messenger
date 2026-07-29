package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort    string
	KafkaBrokers  string
	KafkaInTopic  string
	KafkaOutTopic string
	KafkaGroupID  string
	ScyllaHosts   string
	LogLevel      string
}

func Load() (Config, error) {

	_ = godotenv.Load()

	serverPort, err := env.Get("SERVER_PORT")
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

	kafkaBrokers, err := env.Get("KAFKA_BROKERS")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	kafkaGroupID, err := env.Get("KAFKA_GROUP_ID")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	scyllaHosts, err := env.Get("SCYLLA_HOSTS")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env: %w", err)
	}

	return Config{
		ServerPort:    serverPort,
		KafkaBrokers:  kafkaBrokers,
		KafkaGroupID:  kafkaGroupID,
		KafkaInTopic:  kafkaInTopic,
		KafkaOutTopic: kafkaOutTopic,
		ScyllaHosts:   scyllaHosts,
		LogLevel:      env.GetOr("LOG_LEVEL", "INFO"),
	}, nil
}
