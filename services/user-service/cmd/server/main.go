package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/app"
)

func main() {
	a, err := app.New(context.Background())
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err = a.Start(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started")
	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = a.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown server", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
