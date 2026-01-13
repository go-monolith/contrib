package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/go-monolith/contrib/v1/otel/internal/testutil"
	"github.com/go-monolith/mono/pkg/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetricsRecording(t *testing.T) {
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

	// Record a metric
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, 1, nil)

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find the message count metric
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricMessageCount {
				found = true
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric data type = %T, want Sum[int64]", m.Data)
				}
				if len(sum.DataPoints) == 0 {
					t.Fatal("no data points in metric")
				}
				if sum.DataPoints[0].Value != 1 {
					t.Errorf("metric value = %d, want 1", sum.DataPoints[0].Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric not found")
	}
}

func TestMetricsLabels(t *testing.T) {
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

	// Record a metric with error
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, 1, errors.New("test error"))

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find and verify labels
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricMessageCount {
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric data type = %T, want Sum[int64]", m.Data)
				}
				if len(sum.DataPoints) == 0 {
					t.Fatal("no data points in metric")
				}

				attrs := sum.DataPoints[0].Attributes
				checkAttribute(t, attrs, attrModuleName, "test-module")
				checkAttribute(t, attrs, attrServiceName, "test-service")
				checkAttribute(t, attrs, attrServiceType, serviceTypeRequestReply)
				checkAttribute(t, attrs, attrError, "true")
			}
		}
	}
}

func checkAttribute(t *testing.T, attrs attribute.Set, key, expectedValue string) {
	t.Helper()
	val, ok := attrs.Value(attribute.Key(key))
	if !ok {
		t.Errorf("attribute %q not found", key)
		return
	}
	var actual string
	switch val.Type() {
	case attribute.BOOL:
		if val.AsBool() {
			actual = "true"
		} else {
			actual = "false"
		}
	case attribute.STRING:
		actual = val.AsString()
	default:
		t.Errorf("unsupported attribute type for key %q: %v", key, val.Type())
		return
	}
	if actual != expectedValue {
		t.Errorf("attribute %q = %v, want %q", key, actual, expectedValue)
	}
}

func TestMetricsDisabled(t *testing.T) {
	mw, err := New(WithMetricsDisabled())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mw.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Should not panic when recording metrics
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, 1, nil)

	// Should not panic when recording duration
	mw.recordRequestReplyDuration(context.Background(), "test-module", "test-service", nil, 0.5)
}

func TestAllHandlerTypesMetrics(t *testing.T) {
	tests := []struct {
		name        string
		serviceType string
		setupReg    func(handler func()) types.ServiceRegistration
		callHandler func(reg types.ServiceRegistration)
	}{
		{
			name:        "RequestReply",
			serviceType: serviceTypeRequestReply,
			setupReg: func(handler func()) types.ServiceRegistration {
				return types.ServiceRegistration{
					Name:       "test-service",
					ModuleName: "test-module",
					Type:       types.ServiceTypeRequestReply,
					RequestHandler: func(ctx context.Context, msg *types.Msg) ([]byte, error) {
						handler()
						return []byte("ok"), nil
					},
				}
			},
			callHandler: func(reg types.ServiceRegistration) {
				_, _ = reg.RequestHandler(context.Background(), &types.Msg{})
			},
		},
		{
			name:        "QueueGroup",
			serviceType: serviceTypeQueueGroup,
			setupReg: func(handler func()) types.ServiceRegistration {
				return types.ServiceRegistration{
					Name:       "test-service",
					ModuleName: "test-module",
					Type:       types.ServiceTypeQueueGroup,
					QueueHandlers: []types.QGHP{
						{
							QueueGroup: "test-queue",
							Handler: func(ctx context.Context, msg *types.Msg) error {
								handler()
								return nil
							},
						},
					},
				}
			},
			callHandler: func(reg types.ServiceRegistration) {
				_ = reg.QueueHandlers[0].Handler(context.Background(), &types.Msg{})
			},
		},
		{
			name:        "StreamConsumer",
			serviceType: serviceTypeStreamConsumer,
			setupReg: func(handler func()) types.ServiceRegistration {
				return types.ServiceRegistration{
					Name:       "test-service",
					ModuleName: "test-module",
					Type:       types.ServiceTypeStreamConsumer,
					StreamHandler: func(ctx context.Context, msgs []*types.Msg) error {
						handler()
						return nil
					},
				}
			},
			callHandler: func(reg types.ServiceRegistration) {
				_ = reg.StreamHandler(context.Background(), []*types.Msg{{}})
			},
		},
		{
			name:        "StreamConsumer_MultipleMsgs",
			serviceType: serviceTypeStreamConsumer,
			setupReg: func(handler func()) types.ServiceRegistration {
				return types.ServiceRegistration{
					Name:       "test-service",
					ModuleName: "test-module",
					Type:       types.ServiceTypeStreamConsumer,
					StreamHandler: func(ctx context.Context, msgs []*types.Msg) error {
						handler()
						return nil
					},
				}
			},
			callHandler: func(reg types.ServiceRegistration) {
				_ = reg.StreamHandler(context.Background(), []*types.Msg{{}, {}, {}, {}, {}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			reg := tt.setupReg(func() { called = true })

			result := mw.OnServiceRegistration(context.Background(), reg)
			tt.callHandler(result)

			if !called {
				t.Error("original handler should be called")
			}

			// Verify metrics were recorded
			rm, err := tp.Collect()
			if err != nil {
				t.Fatalf("Collect() error = %v", err)
			}

			found := false
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == metricMessageCount {
						found = true
					}
				}
			}
			if !found {
				t.Error("metric not recorded")
			}
		})
	}
}

func TestRequestReplyDurationRecording(t *testing.T) {
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

	// Record a duration metric
	mw.recordRequestReplyDuration(context.Background(), "test-module", "test-service", nil, 0.123)

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find the duration metric
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricRequestReplyDuration {
				found = true
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric data type = %T, want Histogram[float64]", m.Data)
				}
				if len(hist.DataPoints) == 0 {
					t.Fatal("no data points in histogram")
				}
				if hist.DataPoints[0].Count != 1 {
					t.Errorf("histogram count = %d, want 1", hist.DataPoints[0].Count)
				}
			}
		}
	}

	if !found {
		t.Error("duration metric not found")
	}
}

func TestRequestReplyDurationLabels(t *testing.T) {
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

	// Record a duration metric
	mw.recordRequestReplyDuration(context.Background(), "test-module", "test-service", nil, 0.5)

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find and verify labels
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricRequestReplyDuration {
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric data type = %T, want Histogram[float64]", m.Data)
				}
				if len(hist.DataPoints) == 0 {
					t.Fatal("no data points in histogram")
				}

				attrs := hist.DataPoints[0].Attributes
				checkAttribute(t, attrs, attrModuleName, "test-module")
				checkAttribute(t, attrs, attrServiceName, "test-service")
				checkAttribute(t, attrs, attrServiceType, serviceTypeRequestReply)
				checkAttribute(t, attrs, attrError, "false")
			}
		}
	}
}

func TestRequestReplyDurationWithError(t *testing.T) {
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

	// Record a duration metric with error
	mw.recordRequestReplyDuration(context.Background(), "test-module", "test-service", errors.New("test error"), 0.25)

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find and verify error attribute
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricRequestReplyDuration {
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric data type = %T, want Histogram[float64]", m.Data)
				}
				if len(hist.DataPoints) == 0 {
					t.Fatal("no data points in histogram")
				}

				attrs := hist.DataPoints[0].Attributes
				checkAttribute(t, attrs, attrError, "true")
			}
		}
	}
}

func TestBatchMessageCounting(t *testing.T) {
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

	// Test StreamConsumer with 5 messages
	reg := types.ServiceRegistration{
		Name:       "test-service",
		ModuleName: "test-module",
		Type:       types.ServiceTypeStreamConsumer,
		StreamHandler: func(ctx context.Context, msgs []*types.Msg) error {
			return nil
		},
	}

	result := mw.OnServiceRegistration(context.Background(), reg)
	_ = result.StreamHandler(context.Background(), []*types.Msg{{}, {}, {}, {}, {}})

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Find the message count metric and verify it's 5
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricMessageCount {
				found = true
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric data type = %T, want Sum[int64]", m.Data)
				}
				if len(sum.DataPoints) == 0 {
					t.Fatal("no data points in metric")
				}
				if sum.DataPoints[0].Value != 5 {
					t.Errorf("metric value = %d, want 5 (number of messages in batch)", sum.DataPoints[0].Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric not found")
	}
}

func TestRequestReplyHandlerRecordsDuration(t *testing.T) {
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

	// Create a request-reply handler that takes some time
	called := false
	reg := types.ServiceRegistration{
		Name:       "test-service",
		ModuleName: "test-module",
		Type:       types.ServiceTypeRequestReply,
		RequestHandler: func(ctx context.Context, msg *types.Msg) ([]byte, error) {
			called = true
			return []byte("ok"), nil
		},
	}

	result := mw.OnServiceRegistration(context.Background(), reg)
	_, _ = result.RequestHandler(context.Background(), &types.Msg{})

	if !called {
		t.Error("original handler should be called")
	}

	// Collect metrics
	rm, err := tp.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Verify both counter and histogram were recorded
	foundCounter := false
	foundHistogram := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricMessageCount {
				foundCounter = true
			}
			if m.Name == metricRequestReplyDuration {
				foundHistogram = true
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric data type = %T, want Histogram[float64]", m.Data)
				}
				if len(hist.DataPoints) == 0 {
					t.Fatal("no data points in histogram")
				}
				if hist.DataPoints[0].Count != 1 {
					t.Errorf("histogram count = %d, want 1", hist.DataPoints[0].Count)
				}
			}
		}
	}

	if !foundCounter {
		t.Error("counter metric not recorded")
	}
	if !foundHistogram {
		t.Error("histogram metric not recorded")
	}
}
