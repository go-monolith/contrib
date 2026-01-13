# OTEL Middleware Example

This example demonstrates how to use the OTEL middleware with a Mono application.

## Features Demonstrated

- **OTEL Providers Setup**: Creating TracerProvider, MeterProvider, and LoggerProvider with stdout exporters
- **OTEL Middleware Configuration**: Enabling metrics, traces, and logs
- **Default Logger Integration**: Using `slog.SetDefault()` to export all slog logs to OTEL
- **Service Instrumentation**: Automatic span creation and metrics recording
- **Trace Context Access**: Using `otel.GetTraceContext()` within handlers

## Project Structure

```
examples/v1/otel/
├── go.mod       # Module definition with dependencies
├── main.go      # Application entry point with:
│                # - OTEL providers setup
│                # - GreetingModule (service provider)
│                # - CallerModule (service consumer)
└── README.md    # This file
```

## Running the Example

```bash
# From the examples/v1/otel directory
cd examples/v1/otel

# Download dependencies
go mod tidy

# Run the example
go run main.go
```

## Expected Output

The example will:

1. Start the Mono application with OTEL middleware
2. Wait 2 seconds, then call the `greet` service
3. Output OTEL traces and metrics to stdout
4. Automatically shutdown after demonstrating the example

You should see output similar to:

```
Application started successfully
The example will automatically send a greeting request in 2 seconds...
{
    "Name": "greeting/greet",
    "SpanContext": {
        "TraceID": "abc123...",
        "SpanID": "def456..."
    },
    "Attributes": [
        {"Key": "mono.module", "Value": "greeting"},
        {"Key": "mono.service", "Value": "greet"},
        {"Key": "mono.service_type", "Value": "request_reply"}
    ]
}
...
Example completed! Check the OTEL output above for traces and metrics.
```

## Code Walkthrough

### 1. Setting Up OTEL Providers

```go
// Create resource with service info
res, _ := resource.Merge(
    resource.Default(),
    resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName("otel-example"),
        semconv.ServiceVersion("1.0.0"),
    ),
)

// Create providers with stdout exporters
tracerProvider, _ := setupTracerProvider(res)
meterProvider, _ := setupMeterProvider(res)
loggerProvider, _ := setupLoggerProvider(res)
```

### 2. Creating OTEL Middleware

```go
// Logger is available immediately after New()
otelMw, _ := otel.New(
    otel.WithTracerProvider(tracerProvider), // enables traces
    otel.WithMeterProvider(meterProvider),   // enables metrics
    otel.WithLoggerProvider(loggerProvider), // enables logs
    otel.WithLogLevel(slog.LevelDebug),      // log level filter
)
```

### 3. Integrating with Mono Application

```go
// Set OTEL-bridged logger as the default slog logger
// This exports all slog logs to OTEL
if otelMw.Logger() != nil {
    slog.SetDefault(otelMw.Logger())
}

// Create Mono application
app, _ := mono.NewMonoApplication(
    mono.WithShutdownTimeout(shutdownTimeout),
)

// Register middleware BEFORE other modules
app.Register(otelMw)
app.Register(NewGreetingModule())
```

### 4. Accessing Trace Context in Handlers

```go
func (m *GreetingModule) handleGreet(ctx context.Context, msg *types.Msg) ([]byte, error) {
    // Get trace context from the current span
    traceID, spanID := otel.GetTraceContext(ctx)

    // Log with trace context for correlation
    m.logger.InfoContext(ctx, "Processing request",
        "trace_id", traceID,
        "span_id", spanID,
    )
    // ...
}
```

## Customization

### Using Different Exporters

Replace the stdout exporters with OTLP exporters for production:

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

// OTLP trace exporter
exporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("localhost:4317"),
    otlptracegrpc.WithInsecure(),
)
```

### Disabling Features

```go
// Only enable metrics
otelMw, _ := otel.New(
    otel.WithMeterProvider(meterProvider),
    otel.WithTracesDisabled(),
    otel.WithLogsDisabled(),
)
```

## Related Documentation

- [OTEL Middleware README](../../../v1/otel/README.md)
- [Mono Framework Documentation](https://github.com/go-monolith/mono)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
