package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	kafkainbound "github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/kafka"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/ws"
	kafkaoutbound "github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/chat-service/internal/config"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type App struct {
	wsServer          *ws.Server
	restServer        *rest.Server
	sentPublisher     *kafkaoutbound.MessagePublisher
	persistedConsumer *kafkainbound.PersistedConsumer
}

func New(cfg config.Config, baseLogger *slog.Logger) (*App, error) {
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	hub := chat.NewHub()

	sentPublisher := kafkaoutbound.NewMessagePublisher(brokers, cfg.KafkaOutTopic)

	persistedConsumer := kafkainbound.NewPersistedConsumer(
		brokers,
		cfg.KafkaInTopic,
		cfg.KafkaGroupID,
		hub,
		baseLogger,
	)

	return &App{
		wsServer:          ws.NewServer(cfg.WSPort, hub, sentPublisher, baseLogger),
		restServer:        rest.NewServer(cfg.RESTPort, hub, baseLogger),
		sentPublisher:     sentPublisher,
		persistedConsumer: persistedConsumer,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	errCh := make(chan error, 3)

	go func() {
		if err := a.wsServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("ws server: %w", err)
		}
	}()

	go func() {
		if err := a.restServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("rest server: %w", err)
		}
	}()

	go func() {
		if err := a.persistedConsumer.Start(ctx); err != nil {
			errCh <- fmt.Errorf("persisted consumer: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	return errors.Join(
		a.wsServer.Shutdown(ctx),
		a.restServer.Shutdown(ctx),
		a.sentPublisher.Close(),
		a.persistedConsumer.Close(),
	)
}
