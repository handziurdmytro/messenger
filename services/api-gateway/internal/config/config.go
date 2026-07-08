package config

import (
	"fmt"

	"github.com/handziurdmytro/messenger/pkg/env"
	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel       string
	ServerPort     string
	AuthServiceURL string
	RoutesPath     string
	Routes         []RouteConfig
}

func Load() (Config, error) {
	_ = godotenv.Load()

	serverPort, err := env.Get("SERVER_PORT")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env variables: %w", err)
	}

	authServiceURL, err := env.Get("AUTH_SERVICE_URL")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load env variables: %w", err)
	}

	routesPath := env.GetOr("ROUTES_PATH", "./config/routes.yaml")

	routes, err := loadRoutes(routesPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load routes: %w", err)
	}

	return Config{
		LogLevel:       env.GetOr("LOG_LEVEL", "INFO"),
		ServerPort:     serverPort,
		AuthServiceURL: authServiceURL,
		RoutesPath:     routesPath,
		Routes:         routes,
	}, nil
}
