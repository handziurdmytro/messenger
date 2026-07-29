package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/inbound/rest"
	kafkaoutbound "github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/auth-service/internal/adapter/outbound/postgres"
	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
)

type App struct {
	server    *rest.Server
	publisher *kafkaoutbound.EventPublisher
}

func New(ctx context.Context, cfg config.Config, baseLogger *slog.Logger) (*App, error) {
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	if err = postgres.Migrate(ctx, pool); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	tokenRepo := postgres.NewTokenRepository(pool)

	privateKeyPEM, err := readFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	tokenSvc, err := auth.NewTokenService(
		privateKeyPEM,
		time.Duration(cfg.AccessTTLMinutes)*time.Minute,
		time.Duration(cfg.RefreshTTLDays)*24*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("init token service: %w", err)
	}

	publisher := kafkaoutbound.NewEventPublisher(
		strings.Split(cfg.KafkaBrokers, ","),
		cfg.KafkaTopic,
	)

	registerUC := usecase.NewRegisterUseCase(userRepo, tokenRepo, tokenSvc, publisher)
	loginUC := usecase.NewLoginUseCase(userRepo, tokenRepo, tokenSvc)
	refreshUC := usecase.NewRefreshUseCase(userRepo, tokenRepo, tokenSvc)

	server := rest.NewServer(cfg.ServerPort, registerUC, loginUC, refreshUC, tokenSvc, baseLogger)

	return &App{
		server:    server,
		publisher: publisher,
	}, nil
}

func (a *App) Start() error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return errors.Join(
		a.server.Shutdown(ctx),
		a.publisher.Close(),
	)
}
