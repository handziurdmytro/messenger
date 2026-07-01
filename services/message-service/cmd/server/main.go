package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/app"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("initializing message-service...")

	application, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		slog.Info("starting application consumers and servers...")
		if err := application.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		slog.Error("application crashed", "error", err)
	case sig := <-stopChan:
		slog.Info("received shutdown signal", "signal", sig.String())
	}

	slog.Info("initiating graceful shutdown...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		if err := application.Shutdown(); err != nil {
			slog.Error("error during app shutdown", "error", err)
		}
		close(done)
	}()

	select {
	case <-done:
		slog.Info("application successfully stopped")
	case <-shutdownCtx.Done():
		slog.Warn("shutdown timeout exceeded, forcing exit")
	}
}
