package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Postgres PostgresConfig `envPrefix:"POSTGRES_"`
	NATS     NATSConfig     `envPrefix:"NATS_"`
	Log      LogConfig      `envPrefix:"LOG_"`
}

type ServerConfig struct {
	Port string `env:"PORT,required,notEmpty"`
}

type PostgresConfig struct {
	DSN string `env:"DSN,required,notEmpty"`

	MaxConns     int32 `env:"MAX_CONNS" envDefault:"10"`
	MinIdleConns int32 `env:"MIN_IDLE_CONNS" envDefault:"2"`

	MaxConnLifetime       time.Duration `env:"MAX_CONN_LIFETIME" envDefault:"30m"`
	MaxConnLifetimeJitter time.Duration `env:"MAX_CONN_LIFETIME_JITTER" envDefault:"5m"`
	MaxConnIdleTime       time.Duration `env:"MAX_CONN_IDLE_TIME" envDefault:"5m"`
	HealthCheckPeriod     time.Duration `env:"HEALTH_CHECK_PERIOD" envDefault:"30s"`
	ConnectTimeout        time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
}

type NATSConfig struct {
	URLs           []string      `env:"URLS,required,notEmpty" envSeparator:","`
	Name           string        `env:"NAME" envDefault:"chat-service"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
	ReconnectWait  time.Duration `env:"RECONNECT_WAIT" envDefault:"2s"`
	MaxReconnects  int           `env:"MAX_RECONNECTS" envDefault:"-1"`
	DrainTimeout   time.Duration `env:"DRAIN_TIMEOUT" envDefault:"10s"`

	Stream   string `env:"STREAM" envDefault:"MESSAGES"`
	Consumer string `env:"CONSUMER" envDefault:"chat-message-created"`
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
	if c.Postgres.MaxConns <= 0 {
		return fmt.Errorf("postgres max conns must be positive: %d", c.Postgres.MaxConns)
	}
	if c.Postgres.MinIdleConns < 0 || c.Postgres.MinIdleConns > c.Postgres.MaxConns {
		return fmt.Errorf("invalid postgres min idle conns: %d", c.Postgres.MinIdleConns)
	}
	if c.NATS.ConnectTimeout <= 0 {
		return fmt.Errorf("nats connect timeout must be positive")
	}
	if c.NATS.ReconnectWait < 0 || c.NATS.DrainTimeout < 0 {
		return fmt.Errorf("nats reconnect wait and drain timeout cannot be negative")
	}
	if c.NATS.MaxReconnects < -1 {
		return fmt.Errorf("invalid nats max reconnects: %d", c.NATS.MaxReconnects)
	}

	return nil
}
