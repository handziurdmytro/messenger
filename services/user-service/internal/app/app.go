package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/handziurdmytro/messenger/pkg/natskit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	natsinbound "github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/nats"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres"
	db "github.com/handziurdmytro/messenger/user-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/user-service/internal/config"
	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type App struct {
	server     *rest.Server
	consumer   *natsinbound.UserCreatedConsumer
	natsClient *natskit.Client
	closeDB    func()
	done       chan struct{}
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	pool, err := pgkit.NewPool(ctx, postgresConfig(cfg.Postgres))
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	maxReconnects := cfg.NATS.MaxReconnects
	natsClient, err := natskit.Connect(ctx, natskit.Config{
		URLs:           cfg.NATS.URLs,
		Name:           cfg.NATS.Name,
		ConnectTimeout: cfg.NATS.ConnectTimeout,
		ReconnectWait:  cfg.NATS.ReconnectWait,
		MaxReconnects:  &maxReconnects,
		DrainTimeout:   cfg.NATS.DrainTimeout,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	consumer, err := natsClient.JS.Consumer(ctx, cfg.NATS.Stream, cfg.NATS.Consumer)
	if err != nil {
		natsClient.Close()
		pool.Close()
		return nil, fmt.Errorf("get user-created consumer: %w", err)
	}

	queries := db.New(pool)
	users := postgres.NewUserRepository(queries)
	contacts := postgres.NewContactRepository(queries)

	server := rest.NewServer(
		cfg.Server.Port,
		usecase.NewGetProfileUseCase(users),
		usecase.NewUpdateProfileUseCase(users),
		usecase.NewSearchUsersUseCase(users),
		usecase.NewAddContactUseCase(users, contacts),
		usecase.NewGetContactsUseCase(contacts),
		log,
	)

	return &App{
		server:     server,
		consumer:   natsinbound.NewUserCreatedConsumer(consumer, natsClient.JS, users, log),
		natsClient: natsClient,
		closeDB:    pool.Close,
		done:       make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	defer close(a.done)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := a.server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("user REST: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		return a.consumer.Run(gCtx)
	})
	return g.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	serverErr := a.server.Shutdown(ctx)

	select {
	case <-a.done:
	case <-ctx.Done():
		a.natsClient.Close()
		a.closeDB()
		return errors.Join(serverErr, ctx.Err())
	}

	drainErr := a.natsClient.Drain()
	a.closeDB()
	return errors.Join(serverErr, drainErr)
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
