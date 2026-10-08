package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/handziurdmytro/messenger/pkg/pgkit"
	kafkainbound "github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/kafka"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/config"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type App struct {
	server   *rest.Server
	consumer *kafkainbound.UserCreatedConsumer
	closeDB  func()
}

func New(
	ctx context.Context,
	cfg config.Config,
	baseLogger *slog.Logger,
) (*App, error) {
	pool, err := pgkit.NewPool(ctx, postgresConfig(cfg.Postgres))
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	success := false
	defer func() {
		if !success {
			pool.Close()
		}
	}()

	queries := db.New(pool)

	userRepo := postgres.NewUserRepository(queries)
	contactRepo := postgres.NewContactRepository(queries)

	consumer := kafkainbound.NewUserCreatedConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
		userRepo,
		baseLogger,
	)

	server := rest.NewServer(
		cfg.Server.Port,
		usecase.NewGetProfileUseCase(userRepo),
		usecase.NewUpdateProfileUseCase(userRepo),
		usecase.NewSearchUsersUseCase(userRepo),
		usecase.NewAddContactUseCase(userRepo, contactRepo),
		usecase.NewGetContactsUseCase(contactRepo),
		baseLogger,
	)

	app := &App{
		server:   server,
		consumer: consumer,
		closeDB:  pool.Close,
	}

	success = true
	return app, nil
}

func (a *App) Start(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := a.server.Start(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("rest server: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		return a.consumer.Start(gCtx)
	})

	return g.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	err := errors.Join(
		a.server.Shutdown(ctx),
		a.consumer.Close(),
	)

	a.closeDB()

	return err
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
