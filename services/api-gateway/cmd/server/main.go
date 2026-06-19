package main

import (
	"log/slog"

	"github.com/handziurdmytro/messenger/api-gateway/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
	}

	_ = a
	slog.Info("app initialized, starting server", "port", a.Config.ServerPort)
}
