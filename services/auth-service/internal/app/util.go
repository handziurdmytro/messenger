package app

import (
	"fmt"
	"os"

	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
)

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return data, nil
}

func postgresConfig(cfg config.PostgresConfig) pgkit.Config {
	return pgkit.Config{
		DSN:                   cfg.DSN,
		MaxConns:              cfg.MaxConns,
		MinIdleConns:          cfg.MinIdleConns,
		MaxConnLifetime:       cfg.MaxConnLifetime,
		MaxConnLifetimeJitter: cfg.MaxConnLifetimeJitter,
		MaxConnIdleTime:       cfg.MaxConnIdleTime,
		HealthCheckPeriod:     cfg.HealthCheckPeriod,
		ConnectTimeout:        cfg.ConnectTimeout,
	}
}
