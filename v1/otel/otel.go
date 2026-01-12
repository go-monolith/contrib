package otel

import (
	"context"
	"log/slog"
	"sync"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Version is the current version of this package.
const Version = "v1.0.0"

// Middleware provides OpenTelemetry instrumentation for Mono applications.
// It implements the mono.MiddlewareModule interface to wrap service handlers
// with metrics, tracing, and log export capabilities.
type Middleware struct {
	config Config
	mu     sync.RWMutex

	// Metrics instruments
	meter          metric.Meter
	messageCounter metric.Int64Counter

	// Tracing
	tracer trace.Tracer

	// Logging
	logger *slog.Logger

	// State
	started bool
}

// Compile-time interface check
var _ mono.MiddlewareModule = (*Middleware)(nil)

// New creates a new OpenTelemetry middleware with the given options.
// The logger is initialized immediately so it can be used with mono.WithLogger()
// before the application starts.
func New(opts ...Option) (*Middleware, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	m := &Middleware{
		config: config,
	}

	// Initialize logger IMMEDIATELY so it's available for mono.WithLogger()
	if err := m.initLogs(); err != nil {
		return nil, err
	}

	return m, nil
}

// Name returns the middleware module name.
func (m *Middleware) Name() string {
	return m.config.Name
}

// Start initializes the OpenTelemetry instruments (metrics and tracing).
// The logger is already initialized in New().
func (m *Middleware) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.initMetrics(); err != nil {
		return err
	}

	if err := m.initTracer(); err != nil {
		return err
	}

	m.started = true
	return nil
}

// Stop performs cleanup.
func (m *Middleware) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.started = false
	return nil
}

// Logger returns the OTEL-bridged slog logger.
// This logger is available immediately after New() and can be used with
// mono.WithLogger() for framework-wide logging to OTEL.
// Returns nil if logs are not enabled.
func (m *Middleware) Logger() *slog.Logger {
	return m.logger
}

// OnModuleLifecycle observes module lifecycle events.
// This middleware passes events through unchanged (no lifecycle tracking).
func (m *Middleware) OnModuleLifecycle(
	_ context.Context,
	event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
	return event
}

// OnServiceRegistration wraps service handlers with instrumentation.
func (m *Middleware) OnServiceRegistration(
	_ context.Context,
	reg types.ServiceRegistration,
) types.ServiceRegistration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return reg
	}

	moduleName := reg.ModuleName
	serviceName := reg.Name

	switch reg.Type {
	case types.ServiceTypeRequestReply:
		if reg.RequestHandler != nil {
			handler := reg.RequestHandler
			// Apply tracing first (outer), then metrics (inner)
			if m.config.TracesEnabled {
				handler = m.wrapRequestReplyHandlerWithTracing(handler, moduleName, serviceName)
			}
			if m.config.MetricsEnabled {
				handler = m.wrapRequestReplyHandlerWithMetrics(handler, moduleName, serviceName)
			}
			reg.RequestHandler = handler
		}

	case types.ServiceTypeQueueGroup:
		if len(reg.QueueHandlers) > 0 {
			wrapped := make([]types.QGHP, len(reg.QueueHandlers))
			for i, pair := range reg.QueueHandlers {
				handler := pair.Handler
				if m.config.TracesEnabled {
					handler = m.wrapQueueGroupHandlerWithTracing(handler, moduleName, serviceName)
				}
				if m.config.MetricsEnabled {
					handler = m.wrapQueueGroupHandlerWithMetrics(handler, moduleName, serviceName)
				}
				wrapped[i] = types.QGHP{
					QueueGroup: pair.QueueGroup,
					Handler:    handler,
				}
			}
			reg.QueueHandlers = wrapped
		}

	case types.ServiceTypeStreamConsumer:
		if reg.StreamHandler != nil {
			handler := reg.StreamHandler
			if m.config.TracesEnabled {
				handler = m.wrapStreamConsumerHandlerWithTracing(handler, moduleName, serviceName)
			}
			if m.config.MetricsEnabled {
				handler = m.wrapStreamConsumerHandlerWithMetrics(handler, moduleName, serviceName)
			}
			reg.StreamHandler = handler
		}
	}

	return reg
}

// OnConfigurationChange passes through configuration events unchanged.
func (m *Middleware) OnConfigurationChange(
	_ context.Context,
	event types.ConfigurationEvent,
) types.ConfigurationEvent {
	return event
}

// OnOutgoingMessage injects trace context into outgoing message headers.
func (m *Middleware) OnOutgoingMessage(
	octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
	m.injectTraceContext(octx.Ctx, octx.Msg)
	return octx
}

// OnEventConsumerRegistration wraps event consumer handlers with instrumentation.
func (m *Middleware) OnEventConsumerRegistration(
	_ context.Context,
	entry types.EventConsumerEntry,
) types.EventConsumerEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started || entry.Handler == nil {
		return entry
	}

	moduleName := entry.Module.Name()
	eventName := entry.EventDef.Name

	handler := entry.Handler
	if m.config.TracesEnabled {
		handler = m.wrapEventConsumerHandlerWithTracing(handler, moduleName, eventName)
	}
	if m.config.MetricsEnabled {
		handler = m.wrapEventConsumerHandlerWithMetrics(handler, moduleName, eventName)
	}
	entry.Handler = handler

	return entry
}

// OnEventStreamConsumerRegistration wraps event stream consumer handlers with instrumentation.
func (m *Middleware) OnEventStreamConsumerRegistration(
	_ context.Context,
	entry types.EventStreamConsumerEntry,
) types.EventStreamConsumerEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started || entry.Handler == nil {
		return entry
	}

	moduleName := entry.Module.Name()
	eventName := entry.EventDef.Name

	handler := entry.Handler
	if m.config.TracesEnabled {
		handler = m.wrapEventStreamConsumerHandlerWithTracing(handler, moduleName, eventName)
	}
	if m.config.MetricsEnabled {
		handler = m.wrapEventStreamConsumerHandlerWithMetrics(handler, moduleName, eventName)
	}
	entry.Handler = handler

	return entry
}
