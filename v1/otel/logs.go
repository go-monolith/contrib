package otel

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// initLogs initializes the OTEL-bridged slog logger.
// This is called during New() so the logger is immediately available
// for use with mono.WithLogger() before the application starts.
func (m *Middleware) initLogs() error {
	if !m.config.LogsEnabled {
		return nil
	}
	if m.config.LoggerProvider == nil {
		return fmt.Errorf("LoggerProvider is required when LogsEnabled is true")
	}

	// Create the otelslog handler with the provided LoggerProvider
	handler := otelslog.NewHandler(
		m.config.Name,
		otelslog.WithLoggerProvider(m.config.LoggerProvider),
	)

	// Apply log level filter
	levelHandler := &levelFilterHandler{
		level:   m.config.LogLevel,
		handler: handler,
	}

	// Create the logger
	m.logger = slog.New(levelHandler)

	return nil
}

// levelFilterHandler filters log records by level before passing them
// to the underlying OTEL handler.
type levelFilterHandler struct {
	level   slog.Level
	handler slog.Handler
}

// Enabled reports whether the handler handles records at the given level.
func (h *levelFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.handler.Enabled(ctx, level)
}

// Handle handles the Record.
func (h *levelFilterHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new handler with the given attributes added.
func (h *levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilterHandler{
		level:   h.level,
		handler: h.handler.WithAttrs(attrs),
	}
}

// WithGroup returns a new handler with the given group name.
func (h *levelFilterHandler) WithGroup(name string) slog.Handler {
	return &levelFilterHandler{
		level:   h.level,
		handler: h.handler.WithGroup(name),
	}
}
