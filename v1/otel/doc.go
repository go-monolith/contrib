// Package otel provides OpenTelemetry instrumentation for the Mono framework.
//
// This package implements a MiddlewareModule that automatically instruments
// Mono applications with OpenTelemetry metrics, traces, and logs.
//
// # Features
//
//   - Metrics: Message count with labels (module_name, service_name, service_type, error)
//   - Traces: Automatic span creation for service handlers with W3C Trace Context propagation
//   - Logs: OTEL-bridged slog logger for exporting logs to OpenTelemetry
//
// # Default Configuration
//
// By default, only metrics are enabled. Traces and logs must be explicitly enabled
// by providing the respective providers.
//
// # Quick Start
//
//	// Create OTEL providers (user responsibility)
//	tracerProvider := sdktrace.NewTracerProvider(...)
//	meterProvider := sdkmetric.NewMeterProvider(...)
//	loggerProvider := sdklog.NewLoggerProvider(...)
//
//	// Create OTEL middleware - logger is immediately available after New()
//	otelMw, err := otel.New(
//	    otel.WithMeterProvider(meterProvider),       // enables metrics
//	    otel.WithTracerProvider(tracerProvider),     // enables traces
//	    otel.WithLoggerProvider(loggerProvider),     // enables logs
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create Mono application with OTEL-bridged logger as framework logger
//	app, err := mono.NewMonoApplication(
//	    mono.WithLogger(otelMw.Logger()),  // Framework logs → OTEL
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Register middleware BEFORE other modules
//	app.Register(otelMw)
//	app.Register(&MyModule{})
//
//	// Start the application
//	app.Start(ctx)
//
// # Metrics
//
// The middleware records a counter metric named "mono.message.count" with the
// following attributes:
//
//   - module_name: The name of the module that owns the service
//   - service_name: The name of the service or event
//   - service_type: One of request_reply, queue_group, stream_consumer, event_consumer, event_stream_consumer
//   - error: "true" or "false" indicating if the handler returned an error
//
// # Traces
//
// When traces are enabled, the middleware:
//
//   - Extracts trace context from incoming message headers (W3C Trace Context format)
//   - Creates spans for each handler invocation with appropriate attributes
//   - Propagates trace context to outgoing messages via OnOutgoingMessage hook
//
// Span names follow the pattern:
//
//   - Services: {module_name}/{service_name}
//   - Events: {module_name}/event:{event_name}
//   - Batch handlers: {module_name}/{service_name} [batch]
//
// # Logs
//
// When logs are enabled, the middleware provides an OTEL-bridged slog logger
// via the Logger() method. This logger can be:
//
//   - Used directly as *slog.Logger
//   - Passed to mono.WithLogger() for framework-wide logging to OTEL
//   - Used with slog.SetDefault() to make it the global logger
//
// The logger respects the configured log level filter (default: Info).
//
// # Helper Functions
//
// The package provides helper functions for accessing trace context:
//
//	traceID := otel.GetTraceID(ctx)   // Get trace ID from context
//	spanID := otel.GetSpanID(ctx)     // Get span ID from context
//	traceID, spanID := otel.GetTraceContext(ctx)  // Get both
package otel // import "github.com/go-monolith/contrib/v1/otel"
