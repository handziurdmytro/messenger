package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/app"
	"github.com/handziurdmytro/messenger/message-service/internal/config"
	"github.com/handziurdmytro/messenger/pkg/logger"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logger.New("message-service", "INFO").Error("failed to load config", logger.KeyError, err)
		os.Exit(1)
	}
	log := logger.New("message-service", cfg.LogLevel)

	a, err := app.New(cfg, log)
	if err != nil {
		log.Error("failed to initialize app", logger.KeyError, err)
		os.Exit(1)
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("starting server...")
		return a.Start(gCtx)
	})

	g.Go(func() error {
		<-gCtx.Done()
		log.Info("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return a.Shutdown(shutdownCtx)
	})

	if err = g.Wait(); err != nil {
		log.Error("server exited with error", logger.KeyError, err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
