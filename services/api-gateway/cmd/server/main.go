package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/api-gateway/internal/app"
	"github.com/handziurdmytro/messenger/api-gateway/internal/config"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.New("api-gateway", "INFO").Error("failed to load config", logger.KeyError, err)
		os.Exit(1)
	}

	log := logger.New("api-gateway", cfg.LogLevel)

	a, err := app.New(cfg, log)
	if err != nil {
		log.Error("failed to initialize app", logger.KeyError, err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := a.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", logger.KeyError, err)
			os.Exit(1)
		}
	}()

	log.Info("server started")
	<-quit
	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Shutdown(ctx); err != nil {
		log.Error("failed to shutdown server", logger.KeyError, err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
