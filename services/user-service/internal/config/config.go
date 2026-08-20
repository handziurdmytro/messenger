package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerPort   string `env:"SERVER_PORT,required"`
	PostgresDSN  string `env:"POSTGRES_DSN,required"`
	KafkaBrokers string `env:"KAFKA_BROKERS,required"`
	KafkaTopic   string `env:"KAFKA_TOPIC,required"`
	KafkaGroupID string `env:"KAFKA_GROUP_ID,required"`
	LogLevel     string `env:"LOG_LEVEL" envDefault:"INFO"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse environment config: %w", err)
	}

	return cfg, nil
}
