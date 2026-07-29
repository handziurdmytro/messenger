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
)

func main() {
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	runCtx, cancelRun := context.WithCancel(context.Background())

	errChan := make(chan error, 1)

	go func() {
		if err := a.Start(runCtx); err != nil {
			errChan <- err
		}
	}()

	log.Info("server started")

	select {
	case err := <-errChan:
		log.Error("application crashed", logger.KeyError, err)
	case sig := <-quit:
		log.Info("received shutdown signal", "signal", sig.String())
	}

	cancelRun()
	log.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = a.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown server", logger.KeyError, err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
