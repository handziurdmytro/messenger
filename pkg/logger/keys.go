package logger

import "log/slog"

const (
	KeyComponent = "component"
	KeyPath      = "path"
	KeyMethod    = "method"
	KeyError     = "error"
	KeyUserID    = "user_id"
	KeyTarget    = "target"
)

func WithComponent(l *slog.Logger, name string) *slog.Logger {
	return l.With(KeyComponent, name)
}
