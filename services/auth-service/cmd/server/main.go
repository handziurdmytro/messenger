package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/app"
	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/pkg/runner"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		logger.New("auth-service", "INFO").Error("failed to load config", logger.KeyError, err)
		os.Exit(1)
	}
	log := logger.New("auth-service", cfg.LogLevel)

	a, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Error("failed to initialize app", logger.KeyError, err)
		os.Exit(1)
	}

	log.Info("starting server...")
	if err = runner.Run(ctx, log, 10*time.Second, a); err != nil {
		log.Error("server exited with error", logger.KeyError, err)
		os.Exit(1)
	}
	log.Info("server stopped")
}
