package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/app"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	application, err := app.New()
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)

	go func() {
		slog.Info("starting application servers...")
		if err := application.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		slog.Error("application emergency stop due to server error", "error", err)
	case sig := <-stop:
		slog.Info("received shutdown signal from OS", "signal", sig.String())
	}

	slog.Info("shutting down servers gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = application.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown servers cleanly", "error", err)
		os.Exit(1)
	}

	slog.Info("application stopped successfully")
}
