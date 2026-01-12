package otel

import (
	"context"

	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// metricMessageCount is the name of the message count metric.
	metricMessageCount = "mono.message.count"

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

	return nil
}

// recordMetrics records metrics for a handler invocation.
func (m *Middleware) recordMetrics(
	ctx context.Context,
	moduleName, serviceName, serviceType string,
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

	m.messageCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// wrapRequestReplyHandlerWithMetrics wraps a RequestReply handler to record metrics.
func (m *Middleware) wrapRequestReplyHandlerWithMetrics(
	original types.RequestReplyHandler,
	moduleName, serviceName string,
) types.RequestReplyHandler {
	return func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		resp, err := original(ctx, msg)
		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeRequestReply, err)
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
		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeQueueGroup, err)
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
		// Record one metric per batch
		m.recordMetrics(ctx, moduleName, serviceName, serviceTypeStreamConsumer, err)
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
		m.recordMetrics(ctx, moduleName, eventName, serviceTypeEventConsumer, err)
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
		// Record one metric per batch
		m.recordMetrics(ctx, moduleName, eventName, serviceTypeEventStreamConsumer, err)
		return err
	}
}
