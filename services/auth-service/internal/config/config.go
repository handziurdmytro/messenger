package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Postgres PostgresConfig `envPrefix:"POSTGRES_"`
	Token    TokenConfig    `envPrefix:"TOKEN_"`
	Kafka    KafkaConfig    `envPrefix:"KAFKA_"`
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

type TokenConfig struct {
	PrivateKeyPath string        `env:"PRIVATE_KEY_PATH,required,notEmpty"`
	AccessTTL      time.Duration `env:"ACCESS_TTL" envDefault:"15m"`
	RefreshTTL     time.Duration `env:"REFRESH_TTL" envDefault:"720h"`
}

type KafkaConfig struct {
	Brokers []string `env:"BROKERS,required,notEmpty"`
	Topic   string   `env:"TOPIC,required,notEmpty"`
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

	if c.Postgres.MinIdleConns < 0 {
		return fmt.Errorf("postgres min idle conns cannot be negative: %d", c.Postgres.MinIdleConns)
	}

	if c.Postgres.MinIdleConns > c.Postgres.MaxConns {
		return fmt.Errorf(
			"postgres min idle conns cannot exceed max conns: %d, %d",
			c.Postgres.MinIdleConns,
			c.Postgres.MaxConns,
		)
	}

	return nil
}
