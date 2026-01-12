// Package testutil provides test helpers for the OTEL middleware tests.
package testutil

import (
	"context"

	"github.com/go-monolith/mono/pkg/types"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestMeterProvider wraps a MeterProvider with a ManualReader for testing.
type TestMeterProvider struct {
	Provider *sdkmetric.MeterProvider
	Reader   *sdkmetric.ManualReader
}

// NewTestMeterProvider creates a MeterProvider with a ManualReader for testing.
func NewTestMeterProvider() *TestMeterProvider {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	return &TestMeterProvider{
		Provider: provider,
		Reader:   reader,
	}
}

// Collect collects all metrics from the provider.
func (t *TestMeterProvider) Collect() (*metricdata.ResourceMetrics, error) {
	rm := &metricdata.ResourceMetrics{}
	err := t.Reader.Collect(context.Background(), rm)
	return rm, err
}

// Shutdown shuts down the provider.
func (t *TestMeterProvider) Shutdown() error {
	return t.Provider.Shutdown(context.Background())
}

// TestTracerProvider wraps a TracerProvider with an InMemoryExporter for testing.
type TestTracerProvider struct {
	Provider *sdktrace.TracerProvider
	Exporter *tracetest.InMemoryExporter
}

// NewTestTracerProvider creates a TracerProvider with an InMemoryExporter for testing.
func NewTestTracerProvider() *TestTracerProvider {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	return &TestTracerProvider{
		Provider: provider,
		Exporter: exporter,
	}
}

// Spans returns all recorded spans.
func (t *TestTracerProvider) Spans() tracetest.SpanStubs {
	return t.Exporter.GetSpans()
}

// Reset clears all recorded spans.
func (t *TestTracerProvider) Reset() {
	t.Exporter.Reset()
}

// Shutdown shuts down the provider.
func (t *TestTracerProvider) Shutdown() error {
	return t.Provider.Shutdown(context.Background())
}

// CreateTestMessage creates a types.Msg for testing with optional headers.
func CreateTestMessage(data []byte, headers map[string]string) *types.Msg {
	var h types.Header
	if len(headers) > 0 {
		h = make(types.Header)
		for k, v := range headers {
			h[k] = []string{v}
		}
	}
	return &types.Msg{
		Data:   data,
		Header: h,
	}
}

// CreateTestMessageWithTraceparent creates a test message with a traceparent header.
func CreateTestMessageWithTraceparent(data []byte, traceparent string) *types.Msg {
	return CreateTestMessage(data, map[string]string{
		"traceparent": traceparent,
	})
}

// ValidTraceparent is a valid W3C traceparent header value for testing.
// Format: version-trace_id-parent_id-trace_flags
const ValidTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

// ValidTraceID is the trace ID from ValidTraceparent.
const ValidTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

// ValidSpanID is the span ID from ValidTraceparent.
const ValidSpanID = "00f067aa0ba902b7"
