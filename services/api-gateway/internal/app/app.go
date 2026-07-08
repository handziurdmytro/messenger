package app

import (
	"context"
	"fmt"
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

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	keyProvider := authclient.NewHTTPClient(cfg.AuthServiceURL)

	pubKeyPEM, err := fetchPublicKeyWithRetry(keyProvider, 5, 2*time.Second)
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
	server := rest.NewServer(cfg.ServerPort, router, authenticator)

	return &App{server: server, config: cfg}, nil
}

func fetchPublicKeyWithRetry(provider *authclient.HTTPClient, attempts int, delay time.Duration) ([]byte, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		key, err := provider.FetchPublicKey(ctx)
		cancel()
		if err == nil {
			return key, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, lastErr
}

func (a *App) Start() error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
