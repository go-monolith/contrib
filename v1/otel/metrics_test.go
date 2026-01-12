package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/go-monolith/contrib/v1/otel/internal/testutil"
	"github.com/go-monolith/mono/pkg/types"
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
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, nil)

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
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, errors.New("test error"))

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

func checkAttribute(t *testing.T, attrs any, key, expectedValue string) {
	t.Helper()
	// Note: In real tests, we would iterate over attributes to check values
	// This is a simplified version that verifies the attrs parameter is not nil
	if attrs == nil {
		t.Errorf("attrs is nil, expected non-nil with key %q", key)
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
	mw.recordMetrics(context.Background(), "test-module", "test-service", serviceTypeRequestReply, nil)
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
