package otel

import (
	"context"

	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Span attribute keys
	spanAttrModule      = "mono.module"
	spanAttrService     = "mono.service"
	spanAttrEvent       = "mono.event"
	spanAttrServiceType = "mono.service_type"
	spanAttrBatchSize   = "mono.batch_size"
)

// initTracer initializes the tracer.
func (m *Middleware) initTracer() error {
	if !m.config.TracesEnabled {
		return nil
	}

	tp := m.config.TracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	tracerName := m.config.TracerName
	if tracerName == "" {
		tracerName = m.config.Name
	}

	m.tracer = tp.Tracer(
		tracerName,
		trace.WithInstrumentationVersion(Version),
	)

	return nil
}

// startSpanFromMessage extracts trace context from message headers and starts a new span.
func (m *Middleware) startSpanFromMessage(
	ctx context.Context,
	msg *types.Msg,
	spanName string,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	if !m.config.TracesEnabled || m.tracer == nil {
		return ctx, nil
	}

	// Extract trace context from headers
	ctx = m.extractTraceContext(ctx, msg)

	// Start span
	ctx, span := m.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

// endSpan finishes a span with error status if applicable.
func (m *Middleware) endSpan(span trace.Span, err error) {
	if span == nil {
		return
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

// wrapRequestReplyHandlerWithTracing wraps a RequestReply handler with tracing.
func (m *Middleware) wrapRequestReplyHandlerWithTracing(
	original types.RequestReplyHandler,
	moduleName, serviceName string,
) types.RequestReplyHandler {
	return func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		spanName := moduleName + "/" + serviceName

		ctx, span := m.startSpanFromMessage(ctx, msg, spanName,
			attribute.String(spanAttrModule, moduleName),
			attribute.String(spanAttrService, serviceName),
			attribute.String(spanAttrServiceType, serviceTypeRequestReply),
		)

		resp, err := original(ctx, msg)

		m.endSpan(span, err)
		return resp, err
	}
}

// wrapQueueGroupHandlerWithTracing wraps a QueueGroup handler with tracing.
func (m *Middleware) wrapQueueGroupHandlerWithTracing(
	original types.QueueGroupHandler,
	moduleName, serviceName string,
) types.QueueGroupHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		spanName := moduleName + "/" + serviceName

		ctx, span := m.startSpanFromMessage(ctx, msg, spanName,
			attribute.String(spanAttrModule, moduleName),
			attribute.String(spanAttrService, serviceName),
			attribute.String(spanAttrServiceType, serviceTypeQueueGroup),
		)

		err := original(ctx, msg)

		m.endSpan(span, err)
		return err
	}
}

// wrapStreamConsumerHandlerWithTracing wraps a StreamConsumer handler with tracing.
func (m *Middleware) wrapStreamConsumerHandlerWithTracing(
	original types.StreamConsumerHandler,
	moduleName, serviceName string,
) types.StreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		// For batch handlers, create a parent span for the batch
		spanName := moduleName + "/" + serviceName + " [batch]"

		var parentCtx context.Context
		var span trace.Span

		// Try to extract context from first message
		if len(msgs) > 0 {
			parentCtx, span = m.startSpanFromMessage(ctx, msgs[0], spanName,
				attribute.String(spanAttrModule, moduleName),
				attribute.String(spanAttrService, serviceName),
				attribute.String(spanAttrServiceType, serviceTypeStreamConsumer),
				attribute.Int(spanAttrBatchSize, len(msgs)),
			)
		} else {
			parentCtx = ctx
		}

		err := original(parentCtx, msgs)

		m.endSpan(span, err)
		return err
	}
}

// wrapEventConsumerHandlerWithTracing wraps an EventConsumer handler with tracing.
func (m *Middleware) wrapEventConsumerHandlerWithTracing(
	original types.EventConsumerHandler,
	moduleName, eventName string,
) types.EventConsumerHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		spanName := moduleName + "/event:" + eventName

		ctx, span := m.startSpanFromMessage(ctx, msg, spanName,
			attribute.String(spanAttrModule, moduleName),
			attribute.String(spanAttrEvent, eventName),
			attribute.String(spanAttrServiceType, serviceTypeEventConsumer),
		)

		err := original(ctx, msg)

		m.endSpan(span, err)
		return err
	}
}

// wrapEventStreamConsumerHandlerWithTracing wraps an EventStreamConsumer handler with tracing.
func (m *Middleware) wrapEventStreamConsumerHandlerWithTracing(
	original types.EventStreamConsumerHandler,
	moduleName, eventName string,
) types.EventStreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		spanName := moduleName + "/event:" + eventName + " [batch]"

		var parentCtx context.Context
		var span trace.Span

		if len(msgs) > 0 {
			parentCtx, span = m.startSpanFromMessage(ctx, msgs[0], spanName,
				attribute.String(spanAttrModule, moduleName),
				attribute.String(spanAttrEvent, eventName),
				attribute.String(spanAttrServiceType, serviceTypeEventStreamConsumer),
				attribute.Int(spanAttrBatchSize, len(msgs)),
			)
		} else {
			parentCtx = ctx
		}

		err := original(parentCtx, msgs)

		m.endSpan(span, err)
		return err
	}
}
