package jwt

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
	"github.com/go-monolith/mono/pkg/types"
)

// TestMiddleware_BackgroundRefresh_StartsAndRefreshes tests that the background
// refresh goroutine starts and performs periodic refreshes.
func TestMiddleware_BackgroundRefresh_StartsAndRefreshes(t *testing.T) {
	// Track refresh count
	var refreshCount int
	var refreshMu sync.Mutex

	// Create mock JWKS server
	_, publicKey := testutil.GenerateRSATestKeyPair()
	server := testutil.CreateMockJWKSServer([]*testutil.MockJWKSKey{
		{Kid: "key1", PublicKey: publicKey},
	})
	defer server.Close()

	// Wrap handler to count refreshes
	originalHandler := server.Config.Handler
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshMu.Lock()
		refreshCount++
		refreshMu.Unlock()
		originalHandler.ServeHTTP(w, r)
	})

	// Create JWKS provider
	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Create validator
	config := &Config{
		JWKSRefreshInterval: 100 * time.Millisecond, // Fast refresh for testing
		ClockSkew:           1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config)

	// Create middleware with background refresh enabled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	middleware := &Middleware{
		name:          "jwt",
		config:        config,
		validator:     validator,
		jwksCache:     cache,
		logger:        slog.Default(),
		refreshCtx:    ctx,
		refreshCancel: cancel,
	}

	// Start background refresh in goroutine
	go middleware.startBackgroundRefresh()

	// Wait for at least 3 refreshes (initial state + 3 ticks)
	// With 100ms interval, this should take ~300ms
	time.Sleep(350 * time.Millisecond)

	// Stop the goroutine
	cancel()

	// Give goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// Check that we got multiple refreshes
	refreshMu.Lock()
	count := refreshCount
	refreshMu.Unlock()

	if count < 3 {
		t.Errorf("Expected at least 3 refreshes, got %d", count)
	}
}

// TestMiddleware_BackgroundRefresh_StopsOnCancel tests that the background
// refresh goroutine stops when the context is cancelled.
func TestMiddleware_BackgroundRefresh_StopsOnCancel(t *testing.T) {
	// Track refresh count
	var refreshCount int
	var refreshMu sync.Mutex

	// Create mock JWKS server
	_, publicKey := testutil.GenerateRSATestKeyPair()
	server := testutil.CreateMockJWKSServer([]*testutil.MockJWKSKey{
		{Kid: "key1", PublicKey: publicKey},
	})
	defer server.Close()

	// Wrap handler to count refreshes
	originalHandler := server.Config.Handler
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshMu.Lock()
		refreshCount++
		refreshMu.Unlock()
		originalHandler.ServeHTTP(w, r)
	})

	// Create JWKS provider
	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Create validator
	config := &Config{
		JWKSRefreshInterval: 100 * time.Millisecond, // Fast refresh for testing
		ClockSkew:           1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config)

	// Create middleware
	ctx, cancel := context.WithCancel(context.Background())

	middleware := &Middleware{
		name:          "jwt",
		config:        config,
		validator:     validator,
		jwksCache:     cache,
		logger:        slog.Default(),
		refreshCtx:    ctx,
		refreshCancel: cancel,
	}

	// Start background refresh
	go middleware.startBackgroundRefresh()

	// Wait for a few refreshes
	time.Sleep(250 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// Record count at cancellation
	refreshMu.Lock()
	countAtCancel := refreshCount
	refreshMu.Unlock()

	// Wait longer - no more refreshes should happen
	time.Sleep(300 * time.Millisecond)

	// Check that count hasn't increased
	refreshMu.Lock()
	countAfterCancel := refreshCount
	refreshMu.Unlock()

	if countAfterCancel != countAtCancel {
		t.Errorf("Expected no more refreshes after cancel, but count increased from %d to %d",
			countAtCancel, countAfterCancel)
	}
}

// TestMiddleware_BackgroundRefresh_NoGoroutineLeak tests that the background
// refresh goroutine exits cleanly without leaking.
func TestMiddleware_BackgroundRefresh_NoGoroutineLeak(t *testing.T) {
	// Create mock JWKS server
	_, publicKey := testutil.GenerateRSATestKeyPair()
	server := testutil.CreateMockJWKSServer([]*testutil.MockJWKSKey{
		{Kid: "key1", PublicKey: publicKey},
	})
	defer server.Close()

	// Create JWKS provider
	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Create validator
	config := &Config{
		JWKSRefreshInterval: 50 * time.Millisecond,
		ClockSkew:           1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config)

	// Create a WaitGroup to track goroutine completion
	var wg sync.WaitGroup

	// Create middleware
	ctx, cancel := context.WithCancel(context.Background())

	middleware := &Middleware{
		name:          "jwt",
		config:        config,
		validator:     validator,
		jwksCache:     cache,
		logger:        slog.Default(),
		refreshCtx:    ctx,
		refreshCancel: cancel,
	}

	// Start background refresh with WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		middleware.startBackgroundRefresh()
	}()

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for goroutine to exit with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Goroutine exited cleanly
	case <-time.After(1 * time.Second):
		t.Error("Goroutine did not exit within timeout - potential leak")
	}
}

// TestMiddleware_BackgroundRefresh_RefreshFailure tests that the background
// refresh continues running even when individual refreshes fail.
func TestMiddleware_BackgroundRefresh_RefreshFailure(t *testing.T) {
	// Track refresh attempts
	var attemptCount int
	var attemptMu sync.Mutex

	// Create mock JWKS server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptMu.Lock()
		attemptCount++
		attemptMu.Unlock()

		// Always return 500 error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create JWKS provider
	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Create validator
	config := &Config{
		JWKSRefreshInterval: 100 * time.Millisecond,
		ClockSkew:           1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config)

	// Create middleware
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	middleware := &Middleware{
		name:          "jwt",
		config:        config,
		validator:     validator,
		jwksCache:     cache,
		logger:        slog.Default(),
		refreshCtx:    ctx,
		refreshCancel: cancel,
	}

	// Start background refresh
	go middleware.startBackgroundRefresh()

	// Wait for multiple attempts
	time.Sleep(350 * time.Millisecond)

	// Stop the goroutine
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Check that we got multiple attempts despite failures
	attemptMu.Lock()
	count := attemptCount
	attemptMu.Unlock()

	if count < 3 {
		t.Errorf("Expected at least 3 refresh attempts despite failures, got %d", count)
	}
}

// TestMiddleware_BackgroundRefresh_WithStaticProvider tests that background
// refresh exits gracefully when called with a non-JWKS provider.
func TestMiddleware_BackgroundRefresh_WithStaticProvider(t *testing.T) {
	// Create static provider
	secret := testutil.GenerateHMACTestKey()
	provider := NewStaticKeyProvider(secret)

	// Create validator
	config := &Config{
		JWKSRefreshInterval: 100 * time.Millisecond,
		ClockSkew:           1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config)

	// Create middleware
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	middleware := &Middleware{
		name:          "jwt",
		config:        config,
		validator:     validator,
		jwksCache:     nil, // No cache for static provider
		logger:        slog.Default(),
		refreshCtx:    ctx,
		refreshCancel: cancel,
	}

	// Track goroutine completion
	var wg sync.WaitGroup
	wg.Add(1)

	// Start background refresh
	go func() {
		defer wg.Done()
		middleware.startBackgroundRefresh()
	}()

	// Wait a bit - goroutine should exit on its own when it detects non-JWKS provider
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Goroutine exited as expected
	case <-time.After(500 * time.Millisecond):
		t.Error("Goroutine did not exit when using static provider")
		cancel() // Clean up
	}
}

// TestNew_WithSecret tests middleware creation with HMAC secret.
func TestNew_WithSecret(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithExpectedIssuer("test-issuer"),
		WithRequiredClaims("sub", "email"),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	if mw == nil {
		t.Fatal("Expected middleware instance, got nil")
	}

	if mw.name != "jwt" {
		t.Errorf("Expected name='jwt', got: %s", mw.name)
	}

	if mw.validator == nil {
		t.Error("Expected validator to be initialized")
	}

	if mw.jwksCache != nil {
		t.Error("Expected jwksCache to be nil for static key mode")
	}

	if mw.config.ExpectedIssuer != "test-issuer" {
		t.Errorf("Expected issuer='test-issuer', got: %s", mw.config.ExpectedIssuer)
	}

	if len(mw.config.RequiredClaims) != 2 {
		t.Errorf("Expected 2 required claims, got: %d", len(mw.config.RequiredClaims))
	}
}

// TestNew_WithPublicKey tests middleware creation with RSA public key.
func TestNew_WithPublicKey(t *testing.T) {
	_, publicKey := testutil.GenerateRSATestKeyPair()

	mw, err := New(
		WithPublicKey(publicKey),
		WithClockSkew(2*time.Minute),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	if mw == nil {
		t.Fatal("Expected middleware instance, got nil")
	}

	if mw.validator == nil {
		t.Error("Expected validator to be initialized")
	}

	if mw.jwksCache != nil {
		t.Error("Expected jwksCache to be nil for static key mode")
	}

	if mw.config.ClockSkew != 2*time.Minute {
		t.Errorf("Expected clock skew=2m, got: %v", mw.config.ClockSkew)
	}
}

// TestNew_WithJWKSEndpoint tests middleware creation with JWKS endpoint.
func TestNew_WithJWKSEndpoint(t *testing.T) {
	mw, err := New(
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
		WithJWKSCacheTTL(30*time.Minute),
		WithJWKSRefreshInterval(25*time.Minute),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	if mw == nil {
		t.Fatal("Expected middleware instance, got nil")
	}

	if mw.validator == nil {
		t.Error("Expected validator to be initialized")
	}

	if mw.jwksCache == nil {
		t.Error("Expected jwksCache to be initialized for JWKS mode")
	}

	if mw.config.JWKSEndpoint != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("Expected JWKS endpoint, got: %s", mw.config.JWKSEndpoint)
	}

	if mw.config.JWKSCacheTTL != 30*time.Minute {
		t.Errorf("Expected cache TTL=30m, got: %v", mw.config.JWKSCacheTTL)
	}

	if mw.config.JWKSRefreshInterval != 25*time.Minute {
		t.Errorf("Expected refresh interval=25m, got: %v", mw.config.JWKSRefreshInterval)
	}
}

// TestNew_NoKeySource tests that creation fails when no key source is provided.
func TestNew_NoKeySource(t *testing.T) {
	_, err := New(
		WithExpectedIssuer("test-issuer"),
	)

	if err == nil {
		t.Fatal("Expected error when no key source provided, got nil")
	}

	if err != ErrNoKeySourceConfigured {
		t.Errorf("Expected ErrNoKeySourceConfigured, got: %v", err)
	}
}

// TestNew_MultipleKeySources tests that creation fails when multiple key sources are provided.
func TestNew_MultipleKeySources(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()
	_, publicKey := testutil.GenerateRSATestKeyPair()

	// Test secret + public key
	_, err := New(
		WithSecret(secret),
		WithPublicKey(publicKey),
	)

	if err == nil {
		t.Fatal("Expected error when multiple key sources provided, got nil")
	}

	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("Expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}

	// Test secret + JWKS
	_, err = New(
		WithSecret(secret),
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
	)

	if err == nil {
		t.Fatal("Expected error when multiple key sources provided, got nil")
	}

	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("Expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}

	// Test public key + JWKS
	_, err = New(
		WithPublicKey(publicKey),
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
	)

	if err == nil {
		t.Fatal("Expected error when multiple key sources provided, got nil")
	}

	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("Expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}
}

// TestNew_Defaults tests that default values are applied correctly.
func TestNew_Defaults(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(WithSecret(secret))

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	// Check default values
	if mw.config.ClockSkew != 1*time.Minute {
		t.Errorf("Expected default clock skew=1m, got: %v", mw.config.ClockSkew)
	}

	if mw.config.HeaderKey != "authorization" {
		t.Errorf("Expected default header key='authorization', got: %s", mw.config.HeaderKey)
	}

	if mw.config.TokenPrefix != "Bearer " {
		t.Errorf("Expected default token prefix='Bearer ', got: %s", mw.config.TokenPrefix)
	}

	if mw.config.Optional != false {
		t.Errorf("Expected default optional=false, got: %v", mw.config.Optional)
	}
}

// TestNew_JWKSDefaults tests that JWKS-specific defaults are applied correctly.
func TestNew_JWKSDefaults(t *testing.T) {
	mw, err := New(
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	// Check JWKS default values
	if mw.config.JWKSCacheTTL != 1*time.Hour {
		t.Errorf("Expected default JWKS cache TTL=1h, got: %v", mw.config.JWKSCacheTTL)
	}

	if mw.config.JWKSRefreshInterval != 50*time.Minute {
		t.Errorf("Expected default JWKS refresh interval=50m, got: %v", mw.config.JWKSRefreshInterval)
	}

	if mw.config.JWKSRequestTimeout != 10*time.Second {
		t.Errorf("Expected default JWKS request timeout=10s, got: %v", mw.config.JWKSRequestTimeout)
	}
}

// TestNew_CustomConfiguration tests that custom configuration values override defaults.
func TestNew_CustomConfiguration(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithClockSkew(5*time.Minute),
		WithExpectedIssuer("custom-issuer"),
		WithExpectedAudience("aud1", "aud2"),
		WithRequiredClaims("sub", "email", "role"),
		WithAllowedAlgorithms("HS256", "HS384"),
		WithSkipPaths("health.check", "metrics.prometheus"),
		WithOptional(true),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	// Verify custom values
	if mw.config.ClockSkew != 5*time.Minute {
		t.Errorf("Expected clock skew=5m, got: %v", mw.config.ClockSkew)
	}

	if mw.config.ExpectedIssuer != "custom-issuer" {
		t.Errorf("Expected issuer='custom-issuer', got: %s", mw.config.ExpectedIssuer)
	}

	if len(mw.config.ExpectedAudience) != 2 {
		t.Errorf("Expected 2 audiences, got: %d", len(mw.config.ExpectedAudience))
	}

	if len(mw.config.RequiredClaims) != 3 {
		t.Errorf("Expected 3 required claims, got: %d", len(mw.config.RequiredClaims))
	}

	if len(mw.config.AllowedAlgorithms) != 2 {
		t.Errorf("Expected 2 allowed algorithms, got: %d", len(mw.config.AllowedAlgorithms))
	}

	if len(mw.config.SkipPaths) != 2 {
		t.Errorf("Expected 2 skip paths, got: %d", len(mw.config.SkipPaths))
	}

	if !mw.config.Optional {
		t.Error("Expected optional=true, got: false")
	}
}

// TestNew_ECDSAKey tests middleware creation with ECDSA public key.
func TestNew_ECDSAKey(t *testing.T) {
	_, publicKey := testutil.GenerateECDSATestKeyPair()

	mw, err := New(
		WithPublicKey(publicKey),
	)

	if err != nil {
		t.Fatalf("Expected successful creation, got error: %v", err)
	}

	if mw == nil {
		t.Fatal("Expected middleware instance, got nil")
	}

	if mw.validator == nil {
		t.Error("Expected validator to be initialized")
	}
}

// TestMiddleware_Start_StaticMode tests Start() in static key mode.
func TestMiddleware_Start_StaticMode(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger (normally done by Mono framework)
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify refresh context was created
	if mw.refreshCtx == nil {
		t.Error("Expected refreshCtx to be created")
	}

	if mw.refreshCancel == nil {
		t.Error("Expected refreshCancel to be created")
	}

	// Clean up
	_ = mw.Stop(ctx)
}

// TestMiddleware_Start_JWKSMode tests Start() performs initial JWKS fetch.
func TestMiddleware_Start_JWKSMode(t *testing.T) {
	// Create mock JWKS server
	_, publicKey := testutil.GenerateRSATestKeyPair()
	server := testutil.CreateMockJWKSServer([]*testutil.MockJWKSKey{
		{Kid: "key1", PublicKey: publicKey},
	})
	defer server.Close()

	// Create middleware with JWKS endpoint
	mw, err := New(
		WithJWKSEndpoint(server.URL),
		WithJWKSRefreshInterval(0), // Disable background refresh for this test
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware - should perform initial fetch
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify cache was populated
	key, ok := mw.jwksCache.Get("key1")
	if !ok {
		t.Error("Expected key1 to be in cache after Start()")
	}
	if key == nil {
		t.Error("Expected non-nil key in cache")
	}

	// Clean up
	_ = mw.Stop(ctx)
}

// TestMiddleware_Start_JWKSMode_InitialFetchFails tests Start() fails if initial JWKS fetch fails.
func TestMiddleware_Start_JWKSMode_InitialFetchFails(t *testing.T) {
	// Create middleware with invalid JWKS endpoint
	mw, err := New(
		WithJWKSEndpoint("http://invalid-endpoint-12345.example.com/.well-known/jwks.json"),
		WithJWKSRefreshInterval(0),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start should fail due to initial fetch failure
	ctx := context.Background()
	if err := mw.Start(ctx); err == nil {
		t.Fatal("Expected Start() to fail with invalid JWKS endpoint")
	}

	// Clean up
	_ = mw.Stop(ctx)
}

// TestMiddleware_Start_JWKSMode_BackgroundRefresh tests Start() starts background refresh.
func TestMiddleware_Start_JWKSMode_BackgroundRefresh(t *testing.T) {
	// Track refresh count
	var refreshCount int
	var refreshMu sync.Mutex

	// Create mock JWKS server
	_, publicKey := testutil.GenerateRSATestKeyPair()
	server := testutil.CreateMockJWKSServer([]*testutil.MockJWKSKey{
		{Kid: "key1", PublicKey: publicKey},
	})
	defer server.Close()

	// Wrap handler to count refreshes
	originalHandler := server.Config.Handler
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshMu.Lock()
		refreshCount++
		refreshMu.Unlock()
		originalHandler.ServeHTTP(w, r)
	})

	// Create middleware with background refresh enabled
	mw, err := New(
		WithJWKSEndpoint(server.URL),
		WithJWKSRefreshInterval(100*time.Millisecond), // Fast refresh for testing
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Wait for at least 2 refreshes (initial + 1 background)
	time.Sleep(250 * time.Millisecond)

	// Stop middleware
	if err := mw.Stop(ctx); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Wait a bit for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Check refresh count
	refreshMu.Lock()
	count := refreshCount
	refreshMu.Unlock()

	if count < 2 {
		t.Errorf("Expected at least 2 refreshes (initial + background), got: %d", count)
	}

	// Verify no more refreshes happen after Stop()
	time.Sleep(200 * time.Millisecond)

	refreshMu.Lock()
	finalCount := refreshCount
	refreshMu.Unlock()

	if finalCount != count {
		t.Errorf("Expected no refreshes after Stop(), but count increased from %d to %d", count, finalCount)
	}
}

// TestMiddleware_Stop tests Stop() cancels background refresh.
func TestMiddleware_Stop(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify context is not cancelled yet
	select {
	case <-mw.refreshCtx.Done():
		t.Fatal("refreshCtx should not be cancelled before Stop()")
	default:
		// Expected
	}

	// Stop middleware
	if err := mw.Stop(ctx); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify context was cancelled
	select {
	case <-mw.refreshCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("refreshCtx should be cancelled after Stop()")
	}
}

// TestOnServiceRegistration_RequestReplyHandler tests RequestReply handler wrapping.
func TestOnServiceRegistration_RequestReplyHandler(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Create a test handler (RequestReplyHandler signature: func(ctx, *Msg) ([]byte, error))
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return []byte("response"), nil
	}

	// Create registration
	reg := types.ServiceRegistration{
		Type:           types.ServiceTypeRequestReply,
		Name:           "TestService",
		ModuleName:     "test",
		RequestHandler: originalHandler,
	}

	// Call OnServiceRegistration
	result := mw.OnServiceRegistration(context.Background(), reg)

	// Verify handler was wrapped (address should be different since it's wrapped)
	// Note: Since wrappers are stubs, they return the same handler, so we can't check address
	// But we can verify the registration was processed
	if result.RequestHandler == nil {
		t.Error("Expected RequestHandler to be set")
	}

	if result.Type != types.ServiceTypeRequestReply {
		t.Error("Expected ServiceType to be preserved")
	}
}

// TestOnServiceRegistration_AllHandlerTypes tests all 5 handler types are wrapped.
func TestOnServiceRegistration_AllHandlerTypes(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	ctx := context.Background()

	// Test 1: RequestReply (signature: func(ctx, *Msg) ([]byte, error))
	rrHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		return nil, nil
	}
	rrReg := types.ServiceRegistration{
		Type:           types.ServiceTypeRequestReply,
		Name:           "RRService",
		ModuleName:     "test",
		RequestHandler: rrHandler,
	}
	rrResult := mw.OnServiceRegistration(ctx, rrReg)
	if rrResult.RequestHandler == nil {
		t.Error("Expected RequestReplyHandler to be wrapped")
	}

	// Test 2: QueueGroup
	qgHandler := func(ctx context.Context, msg *types.Msg) error {
		return nil
	}
	qgReg := types.ServiceRegistration{
		Type:       types.ServiceTypeQueueGroup,
		Name:       "QGService",
		ModuleName: "test",
		QueueHandlers: []types.QGHP{
			{QueueGroup: "group1", Handler: qgHandler},
			{QueueGroup: "group2", Handler: qgHandler},
		},
	}
	qgResult := mw.OnServiceRegistration(ctx, qgReg)
	if len(qgResult.QueueHandlers) != 2 {
		t.Error("Expected QueueHandlers to be wrapped")
	}

	// Test 3: StreamConsumer
	scHandler := func(ctx context.Context, msgs []*types.Msg) error {
		return nil
	}
	scReg := types.ServiceRegistration{
		Type:          types.ServiceTypeStreamConsumer,
		Name:          "SCService",
		ModuleName:    "test",
		StreamHandler: scHandler,
	}
	scResult := mw.OnServiceRegistration(ctx, scReg)
	if scResult.StreamHandler == nil {
		t.Error("Expected StreamConsumerHandler to be wrapped")
	}

	// Test 4 & 5: Event consumers are tested via OnEventConsumerRegistration hooks
	// (these are separate tests below)
}

// TestOnServiceRegistration_SkipPaths tests skip paths logic.
func TestOnServiceRegistration_SkipPaths(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	tests := []struct {
		name       string
		skipPaths  []string
		moduleName string
		serviceName string
		shouldSkip bool
	}{
		{
			name:        "exact match module.service",
			skipPaths:   []string{"auth.Login"},
			moduleName:  "auth",
			serviceName: "Login",
			shouldSkip:  true,
		},
		{
			name:        "module wildcard",
			skipPaths:   []string{"auth"},
			moduleName:  "auth",
			serviceName: "GetUser",
			shouldSkip:  true,
		},
		{
			name:        "service wildcard",
			skipPaths:   []string{"Health"},
			moduleName:  "monitoring",
			serviceName: "Health",
			shouldSkip:  true,
		},
		{
			name:        "no match",
			skipPaths:   []string{"auth.Login"},
			moduleName:  "user",
			serviceName: "GetUser",
			shouldSkip:  false,
		},
		{
			name:        "multiple patterns - match second",
			skipPaths:   []string{"auth.Login", "Health"},
			moduleName:  "monitoring",
			serviceName: "Health",
			shouldSkip:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := New(
				WithSecret(secret),
				WithSkipPaths(tt.skipPaths...),
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			// Set logger
			mw.logger = slog.Default()

			// Create registration
			handler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
				return nil, nil
			}
			reg := types.ServiceRegistration{
				Type:           types.ServiceTypeRequestReply,
				Name:           tt.serviceName,
				ModuleName:     tt.moduleName,
				RequestHandler: handler,
			}

			// Call OnServiceRegistration
			result := mw.OnServiceRegistration(context.Background(), reg)

			// For skipped services, handler should remain unchanged
			// For wrapped services, handler is replaced (but stub returns same, so we can't check address)
			// We verify the logic by checking shouldSkip directly
			actualSkip := mw.shouldSkip(tt.moduleName, tt.serviceName)
			if actualSkip != tt.shouldSkip {
				t.Errorf("shouldSkip() = %v, want %v", actualSkip, tt.shouldSkip)
			}

			// Verify registration is always returned (passthrough or modified)
			if result.RequestHandler == nil {
				t.Error("Expected RequestHandler to be set")
			}
		})
	}
}

// TestOnEventConsumerRegistration tests event consumer handler wrapping.
func TestOnEventConsumerRegistration(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Create mock module
	mockModule := &mockModule{name: "testModule"}

	// Create test handler (EventConsumerHandler signature: func(ctx, *Msg) error)
	handler := func(ctx context.Context, msg *types.Msg) error {
		return nil
	}

	// Create event consumer entry
	entry := types.EventConsumerEntry{
		Module: mockModule,
		EventDef: types.BaseEventDefinition{
			ModuleName: "testModule",
			Name:       "TestEvent",
		},
		Handler: handler,
	}

	// Call OnEventConsumerRegistration
	result := mw.OnEventConsumerRegistration(context.Background(), entry)

	// Verify handler was processed
	if result.Handler == nil {
		t.Error("Expected Handler to be set")
	}
}

// TestOnEventStreamConsumerRegistration tests event stream consumer handler wrapping.
func TestOnEventStreamConsumerRegistration(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Create mock module
	mockModule := &mockModule{name: "testModule"}

	// Create test handler (EventStreamConsumerHandler signature: func(ctx, []*Msg) error)
	handler := func(ctx context.Context, msgs []*types.Msg) error {
		return nil
	}

	// Create event stream consumer entry
	entry := types.EventStreamConsumerEntry{
		Module: mockModule,
		EventDef: types.BaseEventDefinition{
			ModuleName: "testModule",
			Name:       "TestEvent",
		},
		Handler: handler,
	}

	// Call OnEventStreamConsumerRegistration
	result := mw.OnEventStreamConsumerRegistration(context.Background(), entry)

	// Verify handler was processed
	if result.Handler == nil {
		t.Error("Expected Handler to be set")
	}
}

// TestOnEventConsumerRegistration_Skip tests skip paths for event consumers.
func TestOnEventConsumerRegistration_Skip(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithSkipPaths("testModule.SkippedEvent"),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Create mock module
	mockModule := &mockModule{name: "testModule"}

	// Create test handler (EventConsumerHandler signature: func(ctx, *Msg) error)
	handler := func(ctx context.Context, msg *types.Msg) error {
		return nil
	}

	// Create event consumer entry for skipped event
	entry := types.EventConsumerEntry{
		Module: mockModule,
		EventDef: types.BaseEventDefinition{
			ModuleName: "testModule",
			Name:       "SkippedEvent",
		},
		Handler: handler,
	}

	// Call OnEventConsumerRegistration
	result := mw.OnEventConsumerRegistration(context.Background(), entry)

	// Verify handler was processed (even when skipped, it's returned)
	if result.Handler == nil {
		t.Error("Expected Handler to be set")
	}

	// Verify skip logic
	if !mw.shouldSkip("testModule", "SkippedEvent") {
		t.Error("Expected event to be skipped")
	}
}

// mockModule is a test helper that implements types.Module
type mockModule struct {
	name string
}

func (m *mockModule) Name() string {
	return m.name
}

func (m *mockModule) Start(ctx context.Context) error {
	return nil
}

func (m *mockModule) Stop(ctx context.Context) error {
	return nil
}

// TestWrapRequestReplyHandler_ValidToken tests that valid token passes through to handler.
func TestWrapRequestReplyHandler_ValidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithExpectedIssuer("test-issuer"),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware to initialize validator
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Track if handler was called
	handlerCalled := false
	var receivedClaims map[string]interface{}

	// Create original handler that checks for claims
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		handlerCalled = true
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Error("Expected claims in context")
		}
		receivedClaims = claims
		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Generate valid token
	token := testutil.GenerateValidJWT(secret, map[string]interface{}{
		"iss": "test-issuer",
		"sub": "user123",
	})

	// Create message with token
	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + token},
		},
		Data: []byte("request"),
	}

	// Call wrapped handler
	response, err := wrappedHandler(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected original handler to be called")
	}

	if string(response) != "success" {
		t.Errorf("Expected response 'success', got: %s", response)
	}

	// Verify claims
	if receivedClaims == nil {
		t.Fatal("Expected claims to be set")
	}

	if receivedClaims["sub"] != "user123" {
		t.Errorf("Expected subject 'user123', got: %v", receivedClaims["sub"])
	}

	if receivedClaims["iss"] != "test-issuer" {
		t.Errorf("Expected issuer 'test-issuer', got: %v", receivedClaims["iss"])
	}
}

// TestWrapRequestReplyHandler_InvalidToken tests that invalid token prevents handler execution.
func TestWrapRequestReplyHandler_InvalidToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Track if handler was called
	handlerCalled := false

	// Create original handler
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		handlerCalled = true
		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Create message with invalid token (wrong signature)
	wrongSecret := testutil.GenerateHMACTestKey()
	invalidToken := testutil.GenerateValidJWT(wrongSecret, nil)

	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + invalidToken},
		},
		Data: []byte("request"),
	}

	// Call wrapped handler
	_, err = wrappedHandler(ctx, msg)
	if err == nil {
		t.Fatal("Expected error for invalid token")
	}

	if handlerCalled {
		t.Error("Expected handler NOT to be called with invalid token")
	}
}

// TestWrapRequestReplyHandler_ClaimsAvailable tests that claims are available in context.
func TestWrapRequestReplyHandler_ClaimsAvailable(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Create original handler that extracts specific claims
	var extractedSub string
	var extractedEmail string
	var extractedRole string

	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, ErrMissingAuthHeader
		}

		extractedSub, _ = claims["sub"].(string)
		extractedEmail, _ = claims["email"].(string)
		extractedRole, _ = claims["role"].(string)

		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Generate token with custom claims
	token := testutil.GenerateValidJWT(secret, map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"role":  "admin",
	})

	// Create message with token
	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + token},
		},
		Data: []byte("request"),
	}

	// Call wrapped handler
	_, err = wrappedHandler(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify custom claims
	if extractedSub != "user123" {
		t.Errorf("Expected sub='user123', got: %s", extractedSub)
	}

	if extractedEmail != "user@example.com" {
		t.Errorf("Expected email='user@example.com', got: %s", extractedEmail)
	}

	if extractedRole != "admin" {
		t.Errorf("Expected role='admin', got: %s", extractedRole)
	}
}

// TestWrapRequestReplyHandler_OptionalMode tests optional mode allows missing token.
func TestWrapRequestReplyHandler_OptionalMode(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		WithOptional(true), // Enable optional mode
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Track if handler was called
	handlerCalled := false
	var receivedClaims map[string]interface{}

	// Create original handler
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		handlerCalled = true
		receivedClaims, _ = ClaimsFromContext(ctx)
		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Create message WITHOUT token
	msg := &types.Msg{
		Header: map[string][]string{},
		Data:   []byte("request"),
	}

	// Call wrapped handler - should succeed in optional mode
	response, err := wrappedHandler(ctx, msg)
	if err != nil {
		t.Fatalf("Expected no error in optional mode, got: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called in optional mode")
	}

	if string(response) != "success" {
		t.Errorf("Expected response 'success', got: %s", response)
	}

	// Verify no claims in context (since no token was provided)
	if receivedClaims != nil {
		t.Error("Expected no claims in context when token is missing")
	}
}

// TestWrapRequestReplyHandler_RequiredMode tests required mode rejects missing token.
func TestWrapRequestReplyHandler_RequiredMode(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
		// Optional defaults to false (required mode)
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Track if handler was called
	handlerCalled := false

	// Create original handler
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		handlerCalled = true
		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Create message WITHOUT token
	msg := &types.Msg{
		Header: map[string][]string{},
		Data:   []byte("request"),
	}

	// Call wrapped handler - should fail in required mode
	_, err = wrappedHandler(ctx, msg)
	if err == nil {
		t.Fatal("Expected error in required mode when token is missing")
	}

	if handlerCalled {
		t.Error("Expected handler NOT to be called when token is missing")
	}

	// Verify error is correct type
	if err != ErrMissingAuthHeader {
		t.Errorf("Expected ErrMissingAuthHeader, got: %v", err)
	}
}

// TestWrapRequestReplyHandler_ExpiredToken tests that expired token is rejected.
func TestWrapRequestReplyHandler_ExpiredToken(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()

	mw, err := New(
		WithSecret(secret),
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Set logger
	mw.logger = slog.Default()

	// Start middleware
	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Failed to start middleware: %v", err)
	}
	defer mw.Stop(ctx)

	// Track if handler was called
	handlerCalled := false

	// Create original handler
	originalHandler := func(ctx context.Context, msg *types.Msg) ([]byte, error) {
		handlerCalled = true
		return []byte("success"), nil
	}

	// Wrap the handler
	wrappedHandler := mw.wrapRequestReplyHandler(originalHandler, "test", "TestService")

	// Generate expired token
	expiredToken := testutil.GenerateExpiredJWT(secret)

	// Create message with expired token
	msg := &types.Msg{
		Header: map[string][]string{
			"Authorization": {"Bearer " + expiredToken},
		},
		Data: []byte("request"),
	}

	// Call wrapped handler - should fail
	_, err = wrappedHandler(ctx, msg)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}

	if handlerCalled {
		t.Error("Expected handler NOT to be called with expired token")
	}
}
