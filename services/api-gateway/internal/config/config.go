package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	LogLevel       string        `env:"LOG_LEVEL" envDefault:"INFO"`
	ServerPort     string        `env:"SERVER_PORT,required"`
	AuthServiceURL string        `env:"AUTH_SERVICE_URL,required"`
	RoutesPath     string        `env:"ROUTES_PATH" envDefault:"./config/routes.yaml"`
	Routes         []RouteConfig `env:"-"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse environment config: %w", err)
	}

	routes, err := loadRoutes(cfg.RoutesPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load routes from %s: %w", cfg.RoutesPath, err)
	}

	cfg.Routes = routes
	return cfg, nil
}
