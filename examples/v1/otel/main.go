// Package main demonstrates how to use the OTEL middleware with a Mono application.
//
// This example shows:
// - Setting up OTEL providers (traces, metrics, logs)
// - Creating the OTEL middleware with all features enabled
// - Using the OTEL-bridged logger for application logging
// - Creating a module with services that are automatically instrumented
// - Accessing trace context within handlers
package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-monolith/contrib/v1/otel"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Create OTEL resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("otel-example"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	// Setup OTEL providers
	tracerProvider, err := setupTracerProvider(res)
	if err != nil {
		log.Fatalf("Failed to create tracer provider: %v", err)
	}

	meterProvider, err := setupMeterProvider(res)
	if err != nil {
		log.Fatalf("Failed to create meter provider: %v", err)
	}

	loggerProvider, err := setupLoggerProvider(res)
	if err != nil {
		log.Fatalf("Failed to create logger provider: %v", err)
	}

	// Create OTEL middleware with all features enabled
	// Logger is available immediately after New()
	otelMw, err := otel.New(
		otel.WithTracerProvider(tracerProvider), // enables traces
		otel.WithMeterProvider(meterProvider),   // enables metrics
		otel.WithLoggerProvider(loggerProvider), // enables logs
		otel.WithLogLevel(slog.LevelDebug),      // log level filter
	)
	if err != nil {
		log.Fatalf("Failed to create OTEL middleware: %v", err)
	}

	// Set OTEL-bridged logger as the default slog logger
	// This exports all slog logs to OTEL
	if otelMw.Logger() != nil {
		slog.SetDefault(otelMw.Logger())
	}

	// Create Mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Register middleware BEFORE other modules
	if err := app.Register(otelMw); err != nil {
		log.Fatalf("Failed to register OTEL middleware: %v", err)
	}

	// Register example modules
	if err := app.Register(NewGreetingModule()); err != nil {
		log.Fatalf("Failed to register greeting module: %v", err)
	}
	if err := app.Register(NewCallerModule()); err != nil {
		log.Fatalf("Failed to register caller module: %v", err)
	}

	// Start
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	log.Println("Application started successfully")
	log.Println("The example will automatically send a greeting request in 2 seconds...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown in order
	if err := app.Stop(shutdownCtx); err != nil {
		log.Printf("Failed to stop app: %v", err)
	}

	// Shutdown OTEL providers in reverse order
	if err := loggerProvider.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown logger provider: %v", err)
	}
	if err := meterProvider.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown meter provider: %v", err)
	}
	if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown tracer provider: %v", err)
	}

	log.Println("Shutdown complete")
}

// setupTracerProvider creates a TracerProvider that exports to stdout.
func setupTracerProvider(res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}

// setupMeterProvider creates a MeterProvider that exports to stdout.
func setupMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)),
		),
		sdkmetric.WithResource(res),
	), nil
}

// setupLoggerProvider creates a LoggerProvider that exports to stdout.
func setupLoggerProvider(res *resource.Resource) (*sdklog.LoggerProvider, error) {
	exporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	), nil
}

// -----------------------------------------------------------------------------
// Greeting Module - provides a greeting service
// -----------------------------------------------------------------------------

// GreetingModule provides a greeting service that responds with personalized greetings.
type GreetingModule struct {
	logger *slog.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*GreetingModule)(nil)
	_ mono.ServiceProviderModule = (*GreetingModule)(nil)
)

// NewGreetingModule creates a new GreetingModule.
func NewGreetingModule() *GreetingModule {
	return &GreetingModule{
		logger: slog.Default(),
	}
}

func (m *GreetingModule) Name() string { return "greeting" }

func (m *GreetingModule) Start(_ context.Context) error {
	m.logger.Info("GreetingModule started")
	return nil
}

func (m *GreetingModule) Stop(_ context.Context) error {
	m.logger.Info("GreetingModule stopped")
	return nil
}

func (m *GreetingModule) RegisterServices(container mono.ServiceContainer) error {
	return container.RegisterRequestReplyService(
		"greet",
		m.handleGreet,
	)
}

// GreetRequest is the request payload for the greet service.
type GreetRequest struct {
	Name string `json:"name"`
}

// GreetResponse is the response payload for the greet service.
type GreetResponse struct {
	Message string `json:"message"`
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
}

func (m *GreetingModule) handleGreet(ctx context.Context, msg *types.Msg) ([]byte, error) {
	// Get trace context from the current span (populated by OTEL middleware)
	traceID, spanID := otel.GetTraceContext(ctx)

	// Log with trace context for correlation
	m.logger.InfoContext(ctx, "Processing greeting request",
		"trace_id", traceID,
		"span_id", spanID,
	)

	var req GreetRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, err
	}

	response := GreetResponse{
		Message: "Hello, " + req.Name + "!",
		TraceID: traceID,
		SpanID:  spanID,
	}

	return json.Marshal(response)
}

// -----------------------------------------------------------------------------
// Caller Module - calls the greeting service
// -----------------------------------------------------------------------------

// CallerModule demonstrates calling the greeting service.
type CallerModule struct {
	logger             *slog.Logger
	greetingContainer  mono.ServiceContainer
	callOnce           sync.Once
	cancelAutoCallFunc context.CancelFunc
}

// Compile-time interface checks
var (
	_ mono.Module          = (*CallerModule)(nil)
	_ mono.DependentModule = (*CallerModule)(nil)
)

// NewCallerModule creates a new CallerModule.
func NewCallerModule() *CallerModule {
	return &CallerModule{
		logger: slog.Default(),
	}
}

func (m *CallerModule) Name() string { return "caller" }

func (m *CallerModule) Start(ctx context.Context) error {
	m.logger.Info("CallerModule started")

	// Start a goroutine to call the greeting service once after a delay
	callCtx, cancel := context.WithCancel(ctx)
	m.cancelAutoCallFunc = cancel

	go m.autoCall(callCtx)

	return nil
}

func (m *CallerModule) Stop(_ context.Context) error {
	if m.cancelAutoCallFunc != nil {
		m.cancelAutoCallFunc()
	}
	m.logger.Info("CallerModule stopped")
	return nil
}

func (m *CallerModule) Dependencies() []string {
	return []string{"greeting"}
}

func (m *CallerModule) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
	if dep == "greeting" {
		m.greetingContainer = container
	}
}

func (m *CallerModule) autoCall(ctx context.Context) {
	// Wait for 2 seconds then call the greeting service
	select {
	case <-ctx.Done():
		return
	case <-time.After(2 * time.Second):
	}

	// Use sync.Once for thread-safe single execution
	m.callOnce.Do(func() {
		m.doCall(ctx)
	})
}

func (m *CallerModule) doCall(ctx context.Context) {
	m.logger.Info("Calling greeting service...")

	// Get the request-reply service client
	greetService, err := m.greetingContainer.GetRequestReplyService("greet")
	if err != nil {
		m.logger.Error("Failed to get greeting service", "error", err)
		return
	}

	// Call the greeting service
	req := GreetRequest{Name: "OTEL User"}
	reqData, err := json.Marshal(req)
	if err != nil {
		m.logger.Error("Failed to marshal request", "error", err)
		return
	}

	respMsg, err := greetService.Call(ctx, reqData)
	if err != nil {
		m.logger.Error("Failed to call greeting service", "error", err)
		return
	}

	var greetResp GreetResponse
	if err := json.Unmarshal(respMsg.Data, &greetResp); err != nil {
		m.logger.Error("Failed to unmarshal response", "error", err)
		return
	}

	m.logger.Info("Received greeting response",
		"message", greetResp.Message,
		"trace_id", greetResp.TraceID,
		"span_id", greetResp.SpanID,
	)

	// Send SIGINT to trigger graceful shutdown after demonstrating the example
	log.Println("\nExample completed! Check the OTEL output above for traces and metrics.")
	log.Println("Shutting down in 3 seconds...")
	time.Sleep(3 * time.Second)

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		log.Printf("Failed to find process: %v", err)
		return
	}
	if err := p.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to send signal: %v", err)
	}
}
