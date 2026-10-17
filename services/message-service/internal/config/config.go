package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server ServerConfig `envPrefix:"SERVER_"`
	Scylla ScyllaConfig `envPrefix:"SCYLLA_"`
	S3     S3Config     `envPrefix:"S3_"`
	NATS   NATSConfig   `envPrefix:"NATS_"`
	Log    LogConfig    `envPrefix:"LOG_"`
}

type ServerConfig struct {
	Port string `env:"PORT,required,notEmpty"`
}

type ScyllaConfig struct {
	Hosts []string `env:"HOSTS,required,notEmpty" envSeparator:","`
}

type S3Config struct {
	Endpoint  string `env:"ENDPOINT,required,notEmpty"`
	Region    string `env:"REGION" envDefault:"us-east-1"`
	Bucket    string `env:"BUCKET,required,notEmpty"`
	AccessKey string `env:"ACCESS_KEY,required,notEmpty"`
	SecretKey string `env:"SECRET_KEY,required,notEmpty"`
}

type NATSConfig struct {
	URLs           []string      `env:"URLS,required,notEmpty" envSeparator:","`
	Name           string        `env:"NAME" envDefault:"message-service"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
	ReconnectWait  time.Duration `env:"RECONNECT_WAIT" envDefault:"2s"`
	MaxReconnects  int           `env:"MAX_RECONNECTS" envDefault:"-1"`
	DrainTimeout   time.Duration `env:"DRAIN_TIMEOUT" envDefault:"10s"`

	Stream   string `env:"STREAM" envDefault:"MESSAGES"`
	Consumer string `env:"CONSUMER" envDefault:"message-send"`
}

type LogConfig struct {
	Level string `env:"LEVEL" envDefault:"INFO"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse environment config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if len(c.Scylla.Hosts) == 0 {
		return fmt.Errorf("scylla hosts are required")
	}
	if len(c.NATS.URLs) == 0 {
		return fmt.Errorf("nats URLs are required")
	}
	if c.NATS.ConnectTimeout <= 0 || c.NATS.ReconnectWait < 0 || c.NATS.DrainTimeout <= 0 {
		return fmt.Errorf("invalid nats timeouts")
	}
	if c.NATS.MaxReconnects < -1 {
		return fmt.Errorf("invalid nats max reconnects: %d", c.NATS.MaxReconnects)
	}
	if c.NATS.Stream == "" || c.NATS.Consumer == "" {
		return fmt.Errorf("nats stream and consumer are required")
	}
	return nil
}
