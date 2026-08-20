package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	WSPort        string `env:"WS_PORT,required"`
	RESTPort      string `env:"REST_PORT,required"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"INFO"`
	KafkaBrokers  string `env:"KAFKA_BROKERS,required"`
	KafkaInTopic  string `env:"KAFKA_IN_TOPIC,required"`
	KafkaOutTopic string `env:"KAFKA_OUT_TOPIC,required"`
	KafkaGroupID  string `env:"KAFKA_GROUP_ID,required"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse environment config: %w", err)
	}

	return cfg, nil
}
