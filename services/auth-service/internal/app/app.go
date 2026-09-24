package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/inbound/rest"
	kafkaoutbound "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres"
	db "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/pgkit"
)

type App struct {
	server    *rest.Server
	publisher *kafkaoutbound.EventPublisher
	closeDB   func()
}

func New(ctx context.Context, cfg config.Config, baseLogger *slog.Logger) (*App, error) {
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
	tokenRepo := postgres.NewTokenRepository(queries)

	privateKeyPEM, err := readFile(cfg.Token.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	tokenSvc, err := auth.NewTokenService(privateKeyPEM, cfg.Token.AccessTTL, cfg.Token.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("init token service: %w", err)
	}

	publisher := kafkaoutbound.NewEventPublisher(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	transactor := postgres.NewTransactor(pool)

	registerUC := usecase.NewRegisterUseCase(transactor, tokenSvc, publisher)
	loginUC := usecase.NewLoginUseCase(userRepo, tokenRepo, tokenSvc)
	refreshUC := usecase.NewRefreshUseCase(userRepo, tokenRepo, transactor, tokenSvc)

	server := rest.NewServer(cfg.Server.Port, registerUC, loginUC, refreshUC, tokenSvc, baseLogger)

	app := &App{
		server:    server,
		publisher: publisher,
		closeDB:   pool.Close,
	}

	success = true
	return app, nil
}

func (a *App) Start(_ context.Context) error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	err := errors.Join(
		a.server.Shutdown(ctx),
		a.publisher.Close(),
	)
	a.closeDB()
	return err
}
