# OTEL Middleware

[![Go Reference](https://pkg.go.dev/badge/github.com/go-monolith/contrib/v1/otel.svg)](https://pkg.go.dev/github.com/go-monolith/contrib/v1/otel)

OpenTelemetry instrumentation middleware for the [Mono Framework](https://github.com/go-monolith/mono).

## Features

- **Metrics**: Message count counter with labels (module_name, service_name, service_type, error)
- **Traces**: Automatic span creation for service handlers with W3C Trace Context propagation
- **Logs**: OTEL-bridged slog logger for exporting logs to OpenTelemetry

## Installation

```bash
go get github.com/go-monolith/contrib/v1/otel
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/contrib/v1/otel"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    sdklog "go.opentelemetry.io/otel/sdk/log"
)

func main() {
    // Create OTEL providers (user responsibility)
    tracerProvider := sdktrace.NewTracerProvider(/* ... */)
    meterProvider := sdkmetric.NewMeterProvider(/* ... */)
    loggerProvider := sdklog.NewLoggerProvider(/* ... */)

    // Create OTEL middleware - logger is immediately available after New()
    otelMw, err := otel.New(
        otel.WithMeterProvider(meterProvider),       // enables metrics
        otel.WithTracerProvider(tracerProvider),     // enables traces
        otel.WithLoggerProvider(loggerProvider),     // enables logs
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create Mono application with OTEL-bridged logger as framework logger
    app, err := mono.NewMonoApplication(
        mono.WithLogger(otelMw.Logger()),  // Framework logs → OTEL
    )
    if err != nil {
        log.Fatal(err)
    }

    // Register middleware BEFORE other modules
    app.Register(otelMw)
    app.Register(&MyModule{})

    // Start the application
    app.Start(context.Background())
}
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithName(string)` | Set middleware name | `"otel"` |
| `WithMeterProvider(metric.MeterProvider)` | Enable metrics with custom provider | Global provider |
| `WithMetricsDisabled()` | Disable metrics collection | Metrics enabled |
| `WithTracerProvider(trace.TracerProvider)` | Enable tracing with custom provider | Disabled |
| `WithTracesDisabled()` | Disable tracing | Traces disabled |
| `WithLoggerProvider(log.LoggerProvider)` | Enable OTEL log export | Disabled |
| `WithLogsDisabled()` | Disable log export | Logs disabled |
| `WithLogLevel(slog.Level)` | Set minimum log level | `slog.LevelInfo` |
| `WithPropagationDisabled()` | Disable trace context propagation | Propagation enabled |

## Metrics

The middleware records a counter metric named `mono.message.count` with the following attributes:

| Attribute | Description |
|-----------|-------------|
| `module_name` | The name of the module that owns the service |
| `service_name` | The name of the service or event |
| `service_type` | One of: `request_reply`, `queue_group`, `stream_consumer`, `event_consumer`, `event_stream_consumer` |
| `error` | Boolean indicating if the handler returned an error |

## Traces

When traces are enabled, the middleware:

- Extracts trace context from incoming message headers (W3C Trace Context format)
- Creates spans for each handler invocation with appropriate attributes
- Propagates trace context to outgoing messages via `OnOutgoingMessage` hook

### Span Naming

| Handler Type | Span Name Pattern |
|--------------|-------------------|
| Services | `{module_name}/{service_name}` |
| Events | `{module_name}/event:{event_name}` |
| Batch handlers | `{module_name}/{service_name} [batch]` |

### Span Attributes

| Attribute | Description |
|-----------|-------------|
| `mono.module` | Module name |
| `mono.service` | Service name |
| `mono.service_type` | Service type |
| `mono.batch_size` | Batch size (for batch handlers) |

## Logs

When logs are enabled, the middleware provides an OTEL-bridged slog logger via the `Logger()` method. This logger can be:

- Used directly as `*slog.Logger`
- Passed to `mono.WithLogger()` for framework-wide logging to OTEL
- Used with `slog.SetDefault()` to make it the global logger

The logger respects the configured log level filter (default: Info).

## Helper Functions

The package provides helper functions for accessing trace context:

```go
traceID := otel.GetTraceID(ctx)              // Get trace ID from context
spanID := otel.GetSpanID(ctx)                // Get span ID from context
traceID, spanID := otel.GetTraceContext(ctx) // Get both
```

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.
