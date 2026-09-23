package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/handziurdmytro/messenger/auth-service/internal/app"
	"github.com/handziurdmytro/messenger/auth-service/internal/config"
	"github.com/handziurdmytro/messenger/pkg/logger"
	"github.com/handziurdmytro/messenger/pkg/runner"
)

// @title           Auth Service API
// @version         1.0
// @description     Authentication and authorization service for messenger.
// @host            localhost:2901
// @BasePath 		/api/auth
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		logger.New("auth-service", "INFO").Error("failed to load config", logger.KeyError, err)
		return 1
	}
	log := logger.New("auth-service", cfg.Log.Level)

	a, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Error("failed to initialize app", logger.KeyError, err)
		return 1
	}

	log.Info("starting server...")
	if err := runner.Run(ctx, log, 10*time.Second, a); err != nil {
		log.Error("server exited with error", logger.KeyError, err)
		return 1
	}
	log.Info("server stopped")
	return 0
}
