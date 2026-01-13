package otel

import (
	"context"

	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HeaderCarrier adapts types.Header to propagation.TextMapCarrier.
// This allows W3C Trace Context headers to be extracted from and injected
// into Mono message headers.
type HeaderCarrier types.Header

// Get returns the value for the given key from the header carrier.
func (c HeaderCarrier) Get(key string) string {
	vals := c[key]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// Set sets the value for the given key in the header carrier.
func (c HeaderCarrier) Set(key, value string) {
	c[key] = []string{value}
}

// Keys returns all keys in the header carrier.
func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// propagator is the W3C TraceContext propagator for trace context propagation.
var propagator = propagation.TraceContext{}

// extractTraceContext extracts trace context from message headers and returns
// a new context with the extracted trace information.
func (m *Middleware) extractTraceContext(ctx context.Context, msg *types.Msg) context.Context {
	if msg == nil || msg.Header == nil {
		return ctx
	}

	carrier := HeaderCarrier(msg.Header)
	return propagator.Extract(ctx, carrier)
}

// injectTraceContext injects trace context from the context into message headers.
// This is called by OnOutgoingMessage to propagate trace context to downstream services.
func (m *Middleware) injectTraceContext(ctx context.Context, msg *types.Msg) {
	if !m.config.PropagationEnabled || !m.config.TracesEnabled {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	if msg.Header == nil {
		msg.Header = make(types.Header)
	}

	carrier := HeaderCarrier(msg.Header)
	propagator.Inject(ctx, carrier)
}

// GetTraceID extracts the trace ID from the context.
// Returns an empty string if no valid trace context exists.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID extracts the span ID from the context.
// Returns an empty string if no valid trace context exists.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// GetTraceContext extracts both trace ID and span ID from the context.
// Returns empty strings if no valid trace context exists.
func GetTraceContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", ""
	}
	sc := span.SpanContext()
	return sc.TraceID().String(), sc.SpanID().String()
}
