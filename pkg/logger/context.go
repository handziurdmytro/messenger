package logger

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type ctxKey string

const (
	loggerKey    ctxKey = "logger"
	requestIDKey ctxKey = "request_id"
)

func WithRequestID(ctx context.Context, base *slog.Logger, requestID string) context.Context {
	if requestID == "" {
		requestID = uuid.New().String()
	}

	enriched := base.With("request_id", requestID)

	ctx = context.WithValue(ctx, requestIDKey, requestID)
	ctx = context.WithValue(ctx, loggerKey, enriched)
	return ctx
}

func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}

func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
