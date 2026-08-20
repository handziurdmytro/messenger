package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/handziurdmytro/messenger/api-gateway/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/api-gateway/internal/adapter/outbound/authclient"
	"github.com/handziurdmytro/messenger/api-gateway/internal/config"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
)

type App struct {
	server *rest.Server
	config config.Config
}

func New(ctx context.Context, cfg config.Config, baseLogger *slog.Logger) (*App, error) {
	keyProvider := authclient.NewHTTPClient(cfg.AuthServiceURL)

	pubKeyPEM, err := fetchPublicKeyWithRetry(ctx, keyProvider, 5, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("fetch public key: %w", err)
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
	server := rest.NewServer(cfg.ServerPort, router, authenticator, baseLogger)

	return &App{server: server, config: cfg}, nil
}

func (a *App) Start(_ context.Context) error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
