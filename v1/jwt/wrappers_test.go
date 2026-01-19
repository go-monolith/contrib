package jwt

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
	"github.com/go-monolith/mono/pkg/types"
)

// Note: TestWrapRequestReplyHandler tests are in jwt_test.go

// TestWrapQueueGroupHandler_ValidToken tests QueueGroup handler with valid token.
func TestWrapQueueGroupHandler_ValidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	// Generate valid JWT
	claims := map[string]interface{}{
		"sub": "user123",
	}
	token := testutil.GenerateValidJWT(secret, claims)

	// Create message with token
	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + token},
		},
	}

	handlerCalled := false
	var receivedClaims map[string]interface{}

	handler := func(ctx context.Context, msg *types.Msg) error {
		handlerCalled = true
		if claims, ok := ClaimsFromContext(ctx); ok {
			receivedClaims = map[string]interface{}(claims)
		}
		return nil
	}

	wrapped := mw.wrapQueueGroupHandler(handler, "test", "TestService")

	err = wrapped(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if receivedClaims == nil {
		t.Error("Expected claims in context")
	} else if receivedClaims["sub"] != "user123" {
		t.Errorf("Expected sub=user123, got: %v", receivedClaims["sub"])
	}
}

// TestWrapQueueGroupHandler_InvalidToken tests QueueGroup handler with invalid token.
func TestWrapQueueGroupHandler_InvalidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer invalid.token.here"},
		},
	}

	handlerCalled := false
	handler := func(ctx context.Context, msg *types.Msg) error {
		handlerCalled = true
		return nil
	}

	wrapped := mw.wrapQueueGroupHandler(handler, "test", "TestService")

	err = wrapped(context.Background(), msg)
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}

	if handlerCalled {
		t.Error("Expected handler NOT to be called with invalid token")
	}
}

// TestWrapStreamConsumerHandler_ValidToken tests StreamConsumer handler with valid token.

// TestWrapStreamConsumerHandler_InvalidToken tests StreamConsumer handler with invalid token.

// TestWrapEventConsumerHandler_ValidToken tests EventConsumer handler with valid token.
func TestWrapEventConsumerHandler_ValidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	// Generate valid JWT
	claims := map[string]interface{}{
		"sub": "user123",
	}
	token := testutil.GenerateValidJWT(secret, claims)

	// Create message with token
	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + token},
		},
	}

	handlerCalled := false
	var receivedClaims map[string]interface{}

	handler := func(ctx context.Context, msg *types.Msg) error {
		handlerCalled = true
		if claims, ok := ClaimsFromContext(ctx); ok {
			receivedClaims = map[string]interface{}(claims)
		}
		return nil
	}

	wrapped := mw.wrapEventConsumerHandler(handler, "test", "TestEvent")

	err = wrapped(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if receivedClaims == nil {
		t.Error("Expected claims in context")
	} else if receivedClaims["sub"] != "user123" {
		t.Errorf("Expected sub=user123, got: %v", receivedClaims["sub"])
	}
}

// TestWrapEventConsumerHandler_InvalidToken tests EventConsumer handler with invalid token.
func TestWrapEventConsumerHandler_InvalidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer invalid.token.here"},
		},
	}

	handlerCalled := false
	handler := func(ctx context.Context, msg *types.Msg) error {
		handlerCalled = true
		return nil
	}

	wrapped := mw.wrapEventConsumerHandler(handler, "test", "TestEvent")

	err = wrapped(context.Background(), msg)
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}

	if handlerCalled {
		t.Error("Expected handler NOT to be called with invalid token")
	}
}

// TestWrapEventStreamConsumerHandler_ValidToken tests EventStreamConsumer handler with valid token.

// TestWrapEventStreamConsumerHandler_InvalidToken tests EventStreamConsumer handler with invalid token.

// TestWrappers_OptionalMode tests that all wrappers respect optional mode.
func TestWrappers_OptionalMode(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithOptional(true),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	// Create message without token
	msg := &types.Msg{
		Header: map[string][]string{},
	}

	// Test RequestReplyHandler
	t.Run("RequestReplyHandler", func(t *testing.T) {
		handlerCalled := false
		handler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
			handlerCalled = true
			return []byte("success"), nil
		}

		wrapped := mw.wrapRequestReplyHandler(handler, "test", "TestService")
		_, err := wrapped(context.Background(), msg)
		if err != nil {
			t.Fatalf("Expected no error in optional mode, got: %v", err)
		}
		if !handlerCalled {
			t.Error("Expected handler to be called in optional mode")
		}
	})

	// Test QueueGroupHandler
	t.Run("QueueGroupHandler", func(t *testing.T) {
		handlerCalled := false
		handler := func(ctx context.Context, msg *types.Msg) error {
			handlerCalled = true
			return nil
		}

		wrapped := mw.wrapQueueGroupHandler(handler, "test", "TestService")
		err := wrapped(context.Background(), msg)
		if err != nil {
			t.Fatalf("Expected no error in optional mode, got: %v", err)
		}
		if !handlerCalled {
			t.Error("Expected handler to be called in optional mode")
		}
	})


	// Test EventConsumerHandler
	t.Run("EventConsumerHandler", func(t *testing.T) {
		handlerCalled := false
		handler := func(ctx context.Context, msg *types.Msg) error {
			handlerCalled = true
			return nil
		}

		wrapped := mw.wrapEventConsumerHandler(handler, "test", "TestEvent")
		err := wrapped(context.Background(), msg)
		if err != nil {
			t.Fatalf("Expected no error in optional mode, got: %v", err)
		}
		if !handlerCalled {
			t.Error("Expected handler to be called in optional mode")
		}
	})

}

// TestWrappers_HandlerErrors tests that handler errors are propagated correctly.
func TestWrappers_HandlerErrors(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}
	mw.logger = slog.Default()

	// Generate valid JWT
	claims := map[string]interface{}{
		"sub": "user123",
	}
	token := testutil.GenerateValidJWT(secret, claims)

	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + token},
		},
	}

	testErr := errors.New("handler error")

	// Test RequestReplyHandler
	t.Run("RequestReplyHandler", func(t *testing.T) {
		handler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
			return nil, testErr
		}

		wrapped := mw.wrapRequestReplyHandler(handler, "test", "TestService")
		_, err := wrapped(context.Background(), msg)
		if err != testErr {
			t.Errorf("Expected handler error to be propagated, got: %v", err)
		}
	})

	// Test QueueGroupHandler
	t.Run("QueueGroupHandler", func(t *testing.T) {
		handler := func(ctx context.Context, msg *types.Msg) error {
			return testErr
		}

		wrapped := mw.wrapQueueGroupHandler(handler, "test", "TestService")
		err := wrapped(context.Background(), msg)
		if err != testErr {
			t.Errorf("Expected handler error to be propagated, got: %v", err)
		}
	})


	// Test EventConsumerHandler
	t.Run("EventConsumerHandler", func(t *testing.T) {
		handler := func(ctx context.Context, msg *types.Msg) error {
			return testErr
		}

		wrapped := mw.wrapEventConsumerHandler(handler, "test", "TestEvent")
		err := wrapped(context.Background(), msg)
		if err != testErr {
			t.Errorf("Expected handler error to be propagated, got: %v", err)
		}
	})

}
