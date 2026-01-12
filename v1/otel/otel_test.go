package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/go-monolith/contrib/v1/otel/internal/testutil"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
)

func TestNew(t *testing.T) {
	t.Run("creates middleware with default config", func(t *testing.T) {
		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if mw == nil {
			t.Fatal("New() returned nil")
		}
		if mw.Name() != "otel" {
			t.Errorf("Name() = %q, want %q", mw.Name(), "otel")
		}
	})

	t.Run("creates middleware with custom name", func(t *testing.T) {
		mw, err := New(WithName("custom-otel"))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if mw.Name() != "custom-otel" {
			t.Errorf("Name() = %q, want %q", mw.Name(), "custom-otel")
		}
	})

	t.Run("metrics enabled by default", func(t *testing.T) {
		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if !mw.config.MetricsEnabled {
			t.Error("MetricsEnabled should be true by default")
		}
	})

	t.Run("traces disabled by default", func(t *testing.T) {
		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if mw.config.TracesEnabled {
			t.Error("TracesEnabled should be false by default")
		}
	})

	t.Run("logs disabled by default", func(t *testing.T) {
		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if mw.config.LogsEnabled {
			t.Error("LogsEnabled should be false by default")
		}
		if mw.Logger() != nil {
			t.Error("Logger() should return nil when logs are disabled")
		}
	})
}

func TestMiddlewareInterface(t *testing.T) {
	// Compile-time check that Middleware implements mono.MiddlewareModule
	var _ mono.MiddlewareModule = (*Middleware)(nil)
}

func TestStartStop(t *testing.T) {
	t.Run("start initializes metrics", func(t *testing.T) {
		tp := testutil.NewTestMeterProvider()
		defer func() { _ = tp.Shutdown() }()

		mw, err := New(WithMeterProvider(tp.Provider))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = mw.Start(context.Background())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		if !mw.started {
			t.Error("middleware should be started")
		}
		if mw.meter == nil {
			t.Error("meter should be initialized")
		}
		if mw.messageCounter == nil {
			t.Error("messageCounter should be initialized")
		}
	})

	t.Run("start initializes tracer", func(t *testing.T) {
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

		if mw.tracer == nil {
			t.Error("tracer should be initialized")
		}
	})

	t.Run("stop clears started flag", func(t *testing.T) {
		mw, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = mw.Start(context.Background())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		err = mw.Stop(context.Background())
		if err != nil {
			t.Fatalf("Stop() error = %v", err)
		}

		if mw.started {
			t.Error("middleware should not be started after Stop()")
		}
	})
}

func TestOnModuleLifecycle(t *testing.T) {
	mw, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	event := types.ModuleLifecycleEvent{
		Type:       types.ModuleStartedEvent,
		ModuleName: "test-module",
	}

	result := mw.OnModuleLifecycle(context.Background(), event)

	// Should pass through unchanged
	if result.Type != event.Type {
		t.Errorf("OnModuleLifecycle() Type = %v, want %v", result.Type, event.Type)
	}
	if result.ModuleName != event.ModuleName {
		t.Errorf("OnModuleLifecycle() ModuleName = %v, want %v", result.ModuleName, event.ModuleName)
	}
}

func TestOnConfigurationChange(t *testing.T) {
	mw, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	event := types.ConfigurationEvent{
		OptionName: "test-option",
		OldValue:   "old-value",
		NewValue:   "new-value",
	}

	result := mw.OnConfigurationChange(context.Background(), event)

	// Should pass through unchanged
	if result.OptionName != event.OptionName {
		t.Errorf("OnConfigurationChange() OptionName = %v, want %v", result.OptionName, event.OptionName)
	}
	if result.NewValue != event.NewValue {
		t.Errorf("OnConfigurationChange() NewValue = %v, want %v", result.NewValue, event.NewValue)
	}
}

func TestOnServiceRegistration(t *testing.T) {
	t.Run("does not wrap handlers before Start", func(t *testing.T) {
		tp := testutil.NewTestMeterProvider()
		defer func() { _ = tp.Shutdown() }()

		mw, err := New(WithMeterProvider(tp.Provider))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		called := false
		originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
			called = true
			return []byte("ok"), nil
		}

		reg := types.ServiceRegistration{
			Name:           "test-service",
			ModuleName:     "test-module",
			Type:           types.ServiceTypeRequestReply,
			RequestHandler: originalHandler,
		}

		result := mw.OnServiceRegistration(context.Background(), reg)

		// Handler should not be wrapped before Start()
		_, _ = result.RequestHandler(context.Background(), &types.Msg{})
		if !called {
			t.Error("original handler should be called")
		}
	})

	t.Run("wraps RequestReply handler after Start", func(t *testing.T) {
		tp := testutil.NewTestMeterProvider()
		defer func() { _ = tp.Shutdown() }()

		mw, err := New(WithMeterProvider(tp.Provider))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = mw.Start(context.Background())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		called := false
		originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
			called = true
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
		resp, err := result.RequestHandler(context.Background(), &types.Msg{})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		if string(resp) != "ok" {
			t.Errorf("handler response = %q, want %q", string(resp), "ok")
		}
		if !called {
			t.Error("original handler should be called")
		}
	})

	t.Run("wraps QueueGroup handler after Start", func(t *testing.T) {
		tp := testutil.NewTestMeterProvider()
		defer func() { _ = tp.Shutdown() }()

		mw, err := New(WithMeterProvider(tp.Provider))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = mw.Start(context.Background())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		called := false
		originalHandler := func(ctx context.Context, msg *types.Msg) error {
			called = true
			return nil
		}

		reg := types.ServiceRegistration{
			Name:       "test-service",
			ModuleName: "test-module",
			Type:       types.ServiceTypeQueueGroup,
			QueueHandlers: []types.QGHP{
				{QueueGroup: "test-queue", Handler: originalHandler},
			},
		}

		result := mw.OnServiceRegistration(context.Background(), reg)

		// Call the wrapped handler
		err = result.QueueHandlers[0].Handler(context.Background(), &types.Msg{})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		if !called {
			t.Error("original handler should be called")
		}
	})

	t.Run("wraps StreamConsumer handler after Start", func(t *testing.T) {
		tp := testutil.NewTestMeterProvider()
		defer func() { _ = tp.Shutdown() }()

		mw, err := New(WithMeterProvider(tp.Provider))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = mw.Start(context.Background())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		called := false
		originalHandler := func(ctx context.Context, msgs []*types.Msg) error {
			called = true
			return nil
		}

		reg := types.ServiceRegistration{
			Name:          "test-service",
			ModuleName:    "test-module",
			Type:          types.ServiceTypeStreamConsumer,
			StreamHandler: originalHandler,
		}

		result := mw.OnServiceRegistration(context.Background(), reg)

		// Call the wrapped handler
		err = result.StreamHandler(context.Background(), []*types.Msg{{}})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		if !called {
			t.Error("original handler should be called")
		}
	})
}

func TestOnServiceRegistrationWithError(t *testing.T) {
	tp := testutil.NewTestMeterProvider()
	defer func() { _ = tp.Shutdown() }()

	mw, err := New(WithMeterProvider(tp.Provider))
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

	// Call the wrapped handler
	_, err = result.RequestHandler(context.Background(), &types.Msg{})
	if !errors.Is(err, expectedErr) {
		t.Errorf("handler error = %v, want %v", err, expectedErr)
	}
}
