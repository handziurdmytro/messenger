package app

import (
	"context"
	"fmt"
	"os"

	"github.com/handziurdmytro/messenger/api-gateway/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/api-gateway/internal/config"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
)

type App struct {
	server *rest.Server
	config config.Config
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	pubKeyPEM, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", cfg.PublicKeyPath, err)
	}

	authenticator, err := auth.NewAuthenticator(pubKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("init authenticator: %w", err)
	}

	routes := make([]proxy.Route, len(cfg.Routes))
	for i, r := range cfg.Routes {
		routes[i] = proxy.Route{
			ID:           proxy.RouteID(r.ID),
			Prefix:       r.Prefix,
			Target:       r.Target,
			StripPrefix:  r.StripPrefix,
			AuthRequired: r.AuthRequired,
		}
	}

	router := proxy.NewRouter(routes)
	server := rest.NewServer(cfg.ServerPort, router, authenticator)

	return &App{
		server: server,
		config: cfg,
	}, nil
}

func (a *App) Start() error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
