package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	natsinbound "github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/nats"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/rest"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/inbound/ws"
	natsoutbound "github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/nats"
	"github.com/handziurdmytro/messenger/chat-service/internal/adapter/outbound/postgres"
	"github.com/handziurdmytro/messenger/chat-service/internal/config"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/handziurdmytro/messenger/pkg/natskit"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

var ErrStartup = errors.New("application startup failed")

type App struct {
	httpServer *rest.Server
	consumer   *natsinbound.MessageConsumer
	registry   *ws.Registry
	natsClient *natskit.Client
	pgPool     *pgxpool.Pool
	done       chan struct{}
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	pgCfg := pgkit.Config{
		DSN:                   cfg.Postgres.DSN,
		MaxConns:              cfg.Postgres.MaxConns,
		MinIdleConns:          cfg.Postgres.MinIdleConns,
		MaxConnLifetime:       cfg.Postgres.MaxConnLifetime,
		MaxConnLifetimeJitter: cfg.Postgres.MaxConnLifetimeJitter,
		MaxConnIdleTime:       cfg.Postgres.MaxConnIdleTime,
		HealthCheckPeriod:     cfg.Postgres.HealthCheckPeriod,
		ConnectTimeout:        cfg.Postgres.ConnectTimeout,
	}

	pool, err := pgkit.NewPool(ctx, pgCfg)
	if err != nil {
		return nil, errkit.Wrap(ErrStartup, "connect postgres", err)
	}

	maxReconnects := cfg.NATS.MaxReconnects
	natsCfg := natskit.Config{
		URLs:           cfg.NATS.URLs,
		Name:           cfg.NATS.Name,
		ConnectTimeout: cfg.NATS.ConnectTimeout,
		ReconnectWait:  cfg.NATS.ReconnectWait,
		MaxReconnects:  &maxReconnects,
		DrainTimeout:   cfg.NATS.DrainTimeout,
	}

	natsClient, err := natskit.Connect(ctx, natsCfg)
	if err != nil {
		pool.Close()
		return nil, errkit.Wrap(ErrStartup, "connect nats", err)
	}

	consumer, err := natsClient.JS.Consumer(ctx, cfg.NATS.Stream, cfg.NATS.Consumer)
	if err != nil {
		natsClient.Close()
		pool.Close()
		return nil, errkit.Wrap(ErrStartup, "get jetstream consumer", err)
	}

	chatRepo := postgres.NewChatRepository(pool)

	createChatUC := usecase.NewCreateChat(chatRepo)
	getChatUC := usecase.NewGetChat(chatRepo)

	publisher := natskit.NewPublisher(natsClient.JS)
	messagePublisher := natsoutbound.NewMessagePublisher(publisher)
	sendMessageUC := usecase.NewSendMessage(chatRepo, messagePublisher)

	registry := ws.NewRegistry()
	wsHandler := ws.NewHandler(registry, sendMessageUC, logger)
	delivery := ws.NewMessageDelivery(chatRepo, registry)

	httpServer := rest.NewServer(
		cfg.Server.Port,
		createChatUC,
		getChatUC,
		wsHandler,
		logger,
		usecase.NewListChats(chatRepo),
	)

	messageConsumer := natsinbound.NewMessageConsumer(
		consumer,
		natsClient.JS,
		delivery,
		logger,
	)

	return &App{
		httpServer: httpServer,
		consumer:   messageConsumer,
		registry:   registry,
		natsClient: natsClient,
		pgPool:     pool,
		done:       make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	defer close(a.done)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := a.httpServer.Start(); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := a.consumer.Run(gCtx); err != nil {
			return fmt.Errorf("nats consumer: %w", err)
		}
		return nil
	})

	return g.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	httpErr := a.httpServer.Shutdown(ctx)
	a.registry.Close()

	select {
	case <-a.done:
	case <-ctx.Done():
		a.natsClient.Close()
		a.pgPool.Close()
		return errors.Join(httpErr, ctx.Err())
	}

	if err := ctx.Err(); err != nil {
		a.natsClient.Close()
		a.pgPool.Close()
		return errors.Join(httpErr, err)
	}

	drainErr := a.natsClient.Drain()
	a.pgPool.Close()

	return errors.Join(httpErr, drainErr)
}
