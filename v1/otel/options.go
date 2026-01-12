package otel

import (
	"log/slog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Option configures the OpenTelemetry middleware.
type Option func(*Config)

// WithName sets the middleware module name.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithMeterProvider sets the OpenTelemetry MeterProvider for metrics.
// This automatically enables metrics collection.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(c *Config) {
		c.MeterProvider = mp
		c.MetricsEnabled = true
	}
}

// WithMetricsDisabled disables metrics collection.
func WithMetricsDisabled() Option {
	return func(c *Config) {
		c.MetricsEnabled = false
	}
}

// WithMeterName sets the name for the OTEL meter.
func WithMeterName(name string) Option {
	return func(c *Config) {
		c.MeterName = name
	}
}

// WithTracerProvider sets the OpenTelemetry TracerProvider for tracing.
// This automatically enables tracing and trace context propagation.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *Config) {
		c.TracerProvider = tp
		c.TracesEnabled = true
		c.PropagationEnabled = true
	}
}

// WithTracesDisabled disables tracing.
func WithTracesDisabled() Option {
	return func(c *Config) {
		c.TracesEnabled = false
	}
}

// WithTracerName sets the name for the OTEL tracer.
func WithTracerName(name string) Option {
	return func(c *Config) {
		c.TracerName = name
	}
}

// WithLoggerProvider sets the OpenTelemetry LoggerProvider for log export.
// This automatically enables log export.
func WithLoggerProvider(lp log.LoggerProvider) Option {
	return func(c *Config) {
		c.LoggerProvider = lp
		c.LogsEnabled = true
	}
}

// WithLogsDisabled disables log export.
func WithLogsDisabled() Option {
	return func(c *Config) {
		c.LogsEnabled = false
	}
}

// WithLogLevel sets the minimum log level to export to OTEL.
func WithLogLevel(level slog.Level) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// WithPropagationDisabled disables automatic trace context propagation
// to outgoing messages.
func WithPropagationDisabled() Option {
	return func(c *Config) {
		c.PropagationEnabled = false
	}
}
