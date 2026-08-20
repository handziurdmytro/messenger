package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/app"
	"github.com/handziurdmytro/messenger/chat-service/internal/config"
	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/pkg/runner"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logger.New("chat-service", "INFO").Error("failed to load config", logger.KeyError, err)
		os.Exit(1)
	}
	log := logger.New("chat-service", cfg.LogLevel)

	a, err := app.New(cfg, log)
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
