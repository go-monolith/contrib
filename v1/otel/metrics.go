package otel

import (
	"context"
	"time"

	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// metricMessageCount is the name of the message count metric.
	metricMessageCount = "mono.message.count"
	// metricRequestReplyDuration is the name of the request-reply duration metric.
	metricRequestReplyDuration = "mono.service.request_reply.duration"

	// Attribute keys
	attrModuleName  = "module_name"
	attrServiceName = "service_name"
	attrServiceType = "service_type"
	attrError       = "error"

	// Service type values
	serviceTypeRequestReply        = "request_reply"
	serviceTypeQueueGroup          = "queue_group"
	serviceTypeStreamConsumer      = "stream_consumer"
	serviceTypeEventConsumer       = "event_consumer"
	serviceTypeEventStreamConsumer = "event_stream_consumer"
)

// initMetrics initializes the metrics instruments.
func (m *Middleware) initMetrics() error {
	if !m.config.MetricsEnabled {
		return nil
	}

	// Use provided MeterProvider or global
	mp := m.config.MeterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	meterName := m.config.MeterName
	if meterName == "" {
		meterName = m.config.Name
	}

	m.meter = mp.Meter(
		meterName,
		metric.WithInstrumentationVersion(Version),
	)

	var err error
	m.messageCounter, err = m.meter.Int64Counter(
		metricMessageCount,
		metric.WithDescription("Number of messages processed by handlers"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return err
	}

	m.requestReplyDuration, err = m.meter.Float64Histogram(
		metricRequestReplyDuration,
		metric.WithDescription("Duration of request-reply handler execution"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	return nil
}

// recordMetrics records metrics for a handler invocation.
// The count parameter represents the number of messages processed in this invocation,
// which is typically 1 for single-message handlers and len(msgs) for batch handlers.
func (m *Middleware) recordMetrics(
	ctx context.Context,
	moduleName, serviceName, serviceType string,
	count int64,
	err error,
) {
	if !m.config.MetricsEnabled || m.messageCounter == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrModuleName, moduleName),
		attribute.String(attrServiceName, serviceName),
		attribute.String(attrServiceType, serviceType),
		attribute.Bool(attrError, err != nil),
	}

	m.messageCounter.Add(ctx, count, metric.WithAttributes(attrs...))
}

// recordRequestReplyDuration records the duration of a request-reply handler invocation.
func (m *Middleware) recordRequestReplyDuration(
	ctx context.Context,
	moduleName, serviceName string,
	err error,
	duration float64,
) {
	if !m.config.MetricsEnabled || m.requestReplyDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrModuleName, moduleName),
		attribute.String(attrServiceName, serviceName),
		attribute.String(attrServiceType, serviceTypeRequestReply),
		attribute.Bool(attrError, err != nil),
	}

	m.requestReplyDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
}

// wrapRequestReplyHandlerWithMetrics wraps a RequestReply handler to record metrics.
func (m *Middleware) wrapRequestReplyHandlerWithMetrics(
	original types.RequestReplyHandler,
	moduleName, serviceName string,
) types.RequestReplyHandler {
	return func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		start := time.Now()
		resp, err := original(ctx, msg)
		duration := time.Since(start).Seconds()

		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeRequestReply, 1, err)
		m.recordRequestReplyDuration(ctx, moduleName, serviceName, err, duration)

		return resp, err
	}
}

// wrapQueueGroupHandlerWithMetrics wraps a QueueGroup handler to record metrics.
func (m *Middleware) wrapQueueGroupHandlerWithMetrics(
	original types.QueueGroupHandler,
	moduleName, serviceName string,
) types.QueueGroupHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		err := original(ctx, msg)
		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeQueueGroup, 1, err)
		return err
	}
}

// wrapStreamConsumerHandlerWithMetrics wraps a StreamConsumer handler to record metrics.
func (m *Middleware) wrapStreamConsumerHandlerWithMetrics(
	original types.StreamConsumerHandler,
	moduleName, serviceName string,
) types.StreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		err := original(ctx, msgs)
		// Record actual number of messages in the batch
		// Note: len(msgs) returns 0 for both nil and empty slices, which correctly
		// represents that no messages were processed
		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeStreamConsumer, int64(len(msgs)), err)
		return err
	}
}

// wrapEventConsumerHandlerWithMetrics wraps an EventConsumer handler to record metrics.
func (m *Middleware) wrapEventConsumerHandlerWithMetrics(
	original types.EventConsumerHandler,
	moduleName, eventName string,
) types.EventConsumerHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		err := original(ctx, msg)
		m.recordMetrics(ctx, moduleName, eventName, serviceTypeEventConsumer, 1, err)
		return err
	}
}

// wrapEventStreamConsumerHandlerWithMetrics wraps an EventStreamConsumer handler to record metrics.
func (m *Middleware) wrapEventStreamConsumerHandlerWithMetrics(
	original types.EventStreamConsumerHandler,
	moduleName, eventName string,
) types.EventStreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		err := original(ctx, msgs)
		// Record actual number of messages in the batch
		// Note: len(msgs) returns 0 for both nil and empty slices, which correctly
		// represents that no messages were processed
		m.recordMetrics(ctx, moduleName, eventName, serviceTypeEventStreamConsumer, int64(len(msgs)), err)
		return err
	}
}
