package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	kafkainbound "github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/kafka"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres"
	"github.com/handziurdmytro/messenger/user-service/internal/config"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type App struct {
	server   *rest.Server
	consumer *kafkainbound.UserCreatedConsumer
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	if err = postgres.Migrate(ctx, pool); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)

	consumer := kafkainbound.NewUserCreatedConsumer(
		strings.Split(cfg.KafkaBrokers, ","),
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
		userRepo,
	)

	server := rest.NewServer(
		cfg.ServerPort,
		usecase.NewGetProfileUseCase(userRepo),
		usecase.NewUpdateProfileUseCase(userRepo),
		usecase.NewSearchUsersUseCase(userRepo),
	)

	return &App{
		server:   server,
		consumer: consumer,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	errch := make(chan error, 2)

	go func() {
		if err := a.server.Start(); !errors.Is(err, http.ErrServerClosed) {
			errch <- fmt.Errorf("rest server: %w", err)
		}
	}()

	go func() {
		if err := a.consumer.Start(ctx); err != nil {
			errch <- fmt.Errorf("consumer: %w", err)
		}
	}()

	return <-errch
}

func (a *App) Shutdown(ctx context.Context) error {
	return errors.Join(
		a.server.Shutdown(ctx),
		a.consumer.Close(),
	)
}
