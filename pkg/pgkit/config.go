package pgkit

import "time"

type Config struct {
	DSN string

	MaxConns     int32
	MinIdleConns int32

	MaxConnLifetime       time.Duration
	MaxConnLifetimeJitter time.Duration
	MaxConnIdleTime       time.Duration
	HealthCheckPeriod     time.Duration

	ConnectTimeout time.Duration
}

func DefaultConfig(dsn string) Config {
	return Config{
		DSN:                   dsn,
		MaxConns:              10,
		MinIdleConns:          2,
		MaxConnLifetime:       30 * time.Minute,
		MaxConnLifetimeJitter: 5 * time.Minute,
		MaxConnIdleTime:       5 * time.Minute,
		HealthCheckPeriod:     30 * time.Second,
		ConnectTimeout:        5 * time.Second,
	}
}
