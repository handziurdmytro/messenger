package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerPort       string `env:"SERVER_PORT,required"`
	PostgresDSN      string `env:"POSTGRES_DSN,required"`
	PrivateKeyPath   string `env:"PRIVATE_KEY_PATH,required"`
	KafkaBrokers     string `env:"KAFKA_BROKERS,required"`
	KafkaTopic       string `env:"KAFKA_TOPIC,required"`
	AccessTTLMinutes int    `env:"ACCESS_TTL_MINUTES" envDefault:"15"`
	RefreshTTLDays   int    `env:"REFRESH_TTL_DAYS" envDefault:"30"`
	LogLevel         string `env:"LOG_LEVEL" envDefault:"INFO"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse environment config: %w", err)
	}

	return cfg, nil
}
