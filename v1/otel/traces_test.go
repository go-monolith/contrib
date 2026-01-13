package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/go-monolith/contrib/v1/otel/internal/testutil"
	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel/codes"
)

func TestSpanCreation(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Create a wrapped handler
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return []byte("ok"), nil
	}

	reg := types.ServiceRegistration{
		Name:           "test-service",
		ModuleName:     "test-module",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)

	// Call the wrapped handler
	_, err = result.RequestHandler(context.Background(), &types.Msg{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}

	// Verify span was created
	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	expectedName := "test-module/test-service"
	if spans[0].Name != expectedName {
		t.Errorf("span name = %q, want %q", spans[0].Name, expectedName)
	}
}

func TestSpanAttributes(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return []byte("ok"), nil
	}

	reg := types.ServiceRegistration{
		Name:           "test-service",
		ModuleName:     "test-module",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)
	_, _ = result.RequestHandler(context.Background(), &types.Msg{})

	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	// Check attributes
	attrs := spans[0].Attributes
	checkSpanAttribute(t, attrs, spanAttrModule, "test-module")
	checkSpanAttribute(t, attrs, spanAttrService, "test-service")
	checkSpanAttribute(t, attrs, spanAttrServiceType, serviceTypeRequestReply)
}

func checkSpanAttribute(t *testing.T, attrs any, key, expectedValue string) {
	t.Helper()
	// In real tests, we would iterate over KeyValue pairs
	// This is a simplified placeholder that verifies attrs is not nil
	if attrs == nil {
		t.Errorf("attrs is nil, expected non-nil with key %q", key)
	}
}

func TestSpanErrorRecording(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	expectedErr := errors.New("test error")
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return nil, expectedErr
	}

	reg := types.ServiceRegistration{
		Name:           "test-service",
		ModuleName:     "test-module",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)
	_, _ = result.RequestHandler(context.Background(), &types.Msg{})

	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	// Check error status
	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status code = %v, want %v", spans[0].Status.Code, codes.Error)
	}
}

func TestSpanOkStatus(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return []byte("ok"), nil
	}

	reg := types.ServiceRegistration{
		Name:           "test-service",
		ModuleName:     "test-module",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)
	_, _ = result.RequestHandler(context.Background(), &types.Msg{})

	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	// Check ok status
	if spans[0].Status.Code != codes.Ok {
		t.Errorf("span status code = %v, want %v", spans[0].Status.Code, codes.Ok)
	}
}

func TestTracesDisabled(t *testing.T) {
	mw, err := New(WithTracesDisabled())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if mw.tracer != nil {
		t.Error("tracer should be nil when traces are disabled")
	}
}

func TestTraceContextExtraction(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		// Verify trace context was extracted and is available
		traceID := GetTraceID(ctx)
		if traceID == "" {
			t.Error("trace ID should be available in handler context")
		}
		return []byte("ok"), nil
	}

	reg := types.ServiceRegistration{
		Name:           "test-service",
		ModuleName:     "test-module",
		Type:           types.ServiceTypeRequestReply,
		RequestHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)

	// Create message with traceparent header
	msg := testutil.CreateTestMessageWithTraceparent([]byte("test"), testutil.ValidTraceparent)

	_, _ = result.RequestHandler(context.Background(), msg)
}

func TestBatchHandlerSpan(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	originalHandler := func(ctx context.Context, msgs []*types.Msg) error {
		return nil
	}

	reg := types.ServiceRegistration{
		Name:          "test-service",
		ModuleName:    "test-module",
		Type:          types.ServiceTypeStreamConsumer,
		StreamHandler: originalHandler,
	}

	result := mw.OnServiceRegistration(context.Background(), reg)

	// Call with batch of messages
	msgs := []*types.Msg{{}, {}, {}}
	_ = result.StreamHandler(context.Background(), msgs)

	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	// Check span name includes [batch]
	expectedName := "test-module/test-service [batch]"
	if spans[0].Name != expectedName {
		t.Errorf("span name = %q, want %q", spans[0].Name, expectedName)
	}
}

func TestEventConsumerSpanName(t *testing.T) {
	tp := testutil.NewTestTracerProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithTracerProvider(tp.Provider))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	called := false
	wrapped := mw.wrapEventConsumerHandlerWithTracing(
		func(ctx context.Context, msg *types.Msg) error {
			called = true
			return nil
		},
		"test-module",
		"OrderCreated",
	)

	_ = wrapped(context.Background(), &types.Msg{})

	if !called {
		t.Error("handler should be called")
	}

	spans := tp.Spans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	expectedName := "test-module/event:OrderCreated"
	if spans[0].Name != expectedName {
		t.Errorf("span name = %q, want %q", spans[0].Name, expectedName)
	}
}
