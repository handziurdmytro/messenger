package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/ws"
	kafka "github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/chat-service/internal/config"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type App struct {
	wsServer   *ws.Server
	restServer *rest.Server
	publisher  *kafka.MessagePublisher
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	hub := chat.NewHub()

	publisher := kafka.NewMessagePublisher(
		strings.Split(cfg.KafkaBrokers, ","),
		cfg.KafkaTopic,
	)

	return &App{
		wsServer:   ws.NewServer(cfg.WSPort, hub, publisher),
		restServer: rest.NewServer(cfg.RESTPort, hub),
		publisher:  publisher,
	}, nil
}

func (a *App) Start() error {
	errch := make(chan error, 2)

	go func() {
		if err := a.wsServer.Start(); !errors.Is(err, http.ErrServerClosed) {
			errch <- fmt.Errorf("ws server: %w", err)
		}
	}()

	go func() {
		if err := a.restServer.Start(); !errors.Is(err, http.ErrServerClosed) {
			errch <- fmt.Errorf("rest server: %w", err)
		}
	}()

	return <-errch
}

func (a *App) Shutdown(ctx context.Context) error {
	wsErr := a.wsServer.Shutdown(ctx)
	restErr := a.restServer.Shutdown(ctx)
	pubErr := a.publisher.Close()
	return errors.Join(wsErr, restErr, pubErr)
}
