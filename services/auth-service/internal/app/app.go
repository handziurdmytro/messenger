package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/inbound/rest"
	natsoutbound "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/nats"
	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres"
	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/natskit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
)

type App struct {
	server     *rest.Server
	worker     *natsoutbound.OutboxWorker
	natsClient *natskit.Client
	closeDB    func()
	done       chan struct{}
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	pool, err := pgkit.NewPool(ctx, postgresConfig(cfg.Postgres))
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	privateKeyPEM, err := readFile(cfg.Token.PrivateKeyPath)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("read private key: %w", err)
	}

	tokenSvc, err := auth.NewTokenService(privateKeyPEM, cfg.Token.AccessTTL, cfg.Token.RefreshTTL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("init token service: %w", err)
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

	queries := db.New(pool)
	users := postgres.NewUserRepository(queries)
	tokens := postgres.NewTokenRepository(queries)
	transactor := postgres.NewTransactor(pool)

	server := rest.NewServer(
		cfg.Server.Port,
		usecase.NewRegisterUseCase(transactor, tokenSvc),
		usecase.NewLoginUseCase(users, tokens, tokenSvc),
		usecase.NewRefreshUseCase(users, tokens, transactor, tokenSvc),
		tokenSvc,
		log,
	)

	return &App{
		server:     server,
		worker:     natsoutbound.NewOutboxWorker(pool, natskit.NewPublisher(natsClient.JS), log, cfg.NATS.OutboxInterval),
		natsClient: natsClient,
		closeDB:    pool.Close,
		done:       make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	defer close(a.done)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := a.server.Start(); err != nil {
			return fmt.Errorf("auth REST: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		return a.worker.Run(gCtx)
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
