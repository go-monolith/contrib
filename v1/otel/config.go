package otel

import (
	"log/slog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the OpenTelemetry middleware configuration.
type Config struct {
	// Name is the middleware module name (default: "otel").
	Name string

	// MetricsEnabled controls whether metrics collection is enabled.
	// Default: true
	MetricsEnabled bool

	// MeterProvider is the OpenTelemetry MeterProvider for metrics.
	// If nil, the global MeterProvider will be used when metrics are enabled.
	MeterProvider metric.MeterProvider

	// MeterName is the name for the OTEL meter.
	// Default: middleware name
	MeterName string

	// TracesEnabled controls whether tracing is enabled.
	// Default: false
	TracesEnabled bool

	// TracerProvider is the OpenTelemetry TracerProvider for tracing.
	// If nil, the global TracerProvider will be used when traces are enabled.
	TracerProvider trace.TracerProvider

	// TracerName is the name for the OTEL tracer.
	// Default: middleware name
	TracerName string

	// LogsEnabled controls whether OTEL log export is enabled.
	// Default: false
	LogsEnabled bool

	// LoggerProvider is the OpenTelemetry LoggerProvider for log export.
	// Required when LogsEnabled is true.
	LoggerProvider log.LoggerProvider

	// LogLevel is the minimum log level to export to OTEL.
	// Default: slog.LevelInfo
	LogLevel slog.Level

	// PropagationEnabled controls whether trace context is automatically
	// propagated to outgoing messages.
	// Default: true (when traces are enabled)
	PropagationEnabled bool
}

// DefaultConfig returns the default configuration for the OTEL middleware.
//
// Example:
//
//	config := otel.DefaultConfig()
//	config.Name = "my-otel"
//	config.TracesEnabled = true
//	// Then create middleware with custom config
func DefaultConfig() Config {
	return Config{
		Name:               "otel",
		MetricsEnabled:     true,
		TracesEnabled:      false,
		LogsEnabled:        false,
		PropagationEnabled: true,
		LogLevel:           slog.LevelInfo,
	}
}
