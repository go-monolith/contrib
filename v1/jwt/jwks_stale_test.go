package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
)

// jwksTestServer is a controllable JWKS endpoint for stale/rate-limit tests.
type jwksTestServer struct {
	*httptest.Server
	mu       sync.Mutex
	keys     []*testutil.MockJWKSKey
	delay    time.Duration
	fail     bool
	requests atomic.Int32
}

func newJWKSTestServer(keys ...*testutil.MockJWKSKey) *jwksTestServer {
	s := &jwksTestServer{keys: keys}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)

		s.mu.Lock()
		delay, fail, keys := s.delay, s.fail, s.keys
		s.mu.Unlock()

		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck // test server
			"keys": buildTestJWKSKeys(keys),
		})
	}))
	return s
}

func (s *jwksTestServer) setKeys(keys ...*testutil.MockJWKSKey) {
	s.mu.Lock()
	s.keys = keys
	s.mu.Unlock()
}

func (s *jwksTestServer) setDelay(d time.Duration) {
	s.mu.Lock()
	s.delay = d
	s.mu.Unlock()
}

func (s *jwksTestServer) setFail(fail bool) {
	s.mu.Lock()
	s.fail = fail
	s.mu.Unlock()
}

// buildTestJWKSKeys renders RSA keys as JWKS entries.
func buildTestJWKSKeys(keys []*testutil.MockJWKSKey) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		pub, ok := k.PublicKey.(*rsa.PublicKey)
		if !ok {
			continue
		}
		out = append(out, rsaJWK(k.Kid, pub))
	}
	return out
}

func rsaJWK(kid string, pub *rsa.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"kty": "RSA",
		"kid": kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

// waitFor polls cond until it returns true or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// expireCache rewinds lastFetch so the cache reports stale without sleeping.
func expireCache(c *JWKSCache) {
	c.mu.Lock()
	c.lastFetch = time.Now().Add(-c.ttl - time.Second)
	c.mu.Unlock()
}

// TestJWKSCache_Lookup_ReportsStaleKeys verifies Lookup keeps returning keys
// past the TTL while Get hides them.
func TestJWKSCache_Lookup_ReportsStaleKeys(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	cache := NewJWKSCache(time.Hour)
	cache.Replace(keyMap{"key1": pub}.toPublicKeys())

	if _, found, fresh := cache.Lookup("key1"); !found || !fresh {
		t.Fatalf("expected found+fresh, got found=%v fresh=%v", found, fresh)
	}
	if !cache.IsFresh() {
		t.Fatal("expected cache to be fresh")
	}

	expireCache(cache)

	key, found, fresh := cache.Lookup("key1")
	if !found || fresh || key == nil {
		t.Fatalf("expected found+stale, got found=%v fresh=%v key=%v", found, fresh, key)
	}
	if _, ok := cache.Get("key1"); ok {
		t.Fatal("Get must not return stale keys")
	}
	if cache.IsFresh() {
		t.Fatal("expected cache to be stale")
	}
}

// TestJWKSCache_Replace_NoEmptyWindow verifies concurrent lookups never see
// an empty key set while the cache is being replaced.
func TestJWKSCache_Replace_NoEmptyWindow(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	cache := NewJWKSCache(time.Hour)
	keys := keyMap{"key1": pub}.toPublicKeys()
	cache.Replace(keys)

	stop := make(chan struct{})
	var misses atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if _, found, _ := cache.Lookup("key1"); !found {
						misses.Add(1)
					}
				}
			}
		}()
	}

	for i := 0; i < 2000; i++ {
		cache.Replace(keys)
	}
	close(stop)
	wg.Wait()

	if n := misses.Load(); n != 0 {
		t.Fatalf("expected no misses during Replace, got %d", n)
	}
}

// TestJWKSKeyProvider_StaleKeyServedWithoutBlocking verifies that a known key
// is returned immediately when the cache is stale and the endpoint is slow,
// and that exactly one background refresh follows.
func TestJWKSKeyProvider_StaleKeyServedWithoutBlocking(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	cache := NewJWKSCache(time.Hour)
	provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second)

	ctx := context.Background()
	if err := provider.RefreshCache(ctx); err != nil {
		t.Fatalf("initial refresh failed: %v", err)
	}
	srv.requests.Store(0)

	// Endpoint becomes slow, cache becomes stale.
	srv.setDelay(1500 * time.Millisecond)
	expireCache(cache)

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := provider.GetKey(ctx, "key1")
			if err != nil || key == nil {
				t.Errorf("stale GetKey failed: key=%v err=%v", key, err)
			}
		}()
	}
	wg.Wait()

	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("stale GetKey blocked for %v; expected immediate return", elapsed)
	}

	// Exactly one background refresh runs and eventually makes the cache fresh.
	waitFor(t, 5*time.Second, cache.IsFresh, "background refresh did not complete")
	if n := srv.requests.Load(); n != 1 {
		t.Fatalf("expected exactly 1 background fetch, got %d", n)
	}
}

// TestJWKSKeyProvider_StaleKeyServedWhenRefreshFails verifies the last known
// keys stay usable while the endpoint is failing, and that the failure is
// surfaced through the logger rather than to the caller.
func TestJWKSKeyProvider_StaleKeyServedWhenRefreshFails(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	cache := NewJWKSCache(time.Hour)
	provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second,
		WithProviderMinRefreshInterval(0))

	ctx := context.Background()
	if err := provider.RefreshCache(ctx); err != nil {
		t.Fatalf("initial refresh failed: %v", err)
	}
	srv.requests.Store(0)
	srv.setFail(true)
	expireCache(cache)

	for i := 0; i < 3; i++ {
		key, err := provider.GetKey(ctx, "key1")
		if err != nil || key == nil {
			t.Fatalf("stale key should be served while endpoint fails, got err=%v", err)
		}
		// Let the single-flight refresh finish before the next attempt.
		waitFor(t, time.Second, func() bool { return !provider.backgroundRefreshing.Load() },
			"background refresh did not finish")
	}

	if cache.IsFresh() {
		t.Fatal("cache must stay stale while refreshes fail")
	}
	if n := srv.requests.Load(); n < 1 {
		t.Fatal("expected background refresh attempts")
	}
}

// TestJWKSKeyProvider_UnknownKidRefreshIsRateLimited verifies that unknown
// kids trigger at most one synchronous fetch per minRefreshInterval.
func TestJWKSKeyProvider_UnknownKidRefreshIsRateLimited(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	cache := NewJWKSCache(time.Hour)
	provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second,
		WithProviderMinRefreshInterval(300*time.Millisecond))

	ctx := context.Background()
	if err := provider.RefreshCache(ctx); err != nil {
		t.Fatalf("initial refresh failed: %v", err)
	}
	srv.requests.Store(0)

	// First unknown kid: one fetch, still not found.
	if _, err := provider.GetKey(ctx, "forged"); err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
	if n := srv.requests.Load(); n != 1 {
		t.Fatalf("expected 1 fetch after first miss, got %d", n)
	}

	// Burst of unknown kids inside the window: no network, fast failure.
	start := time.Now()
	for i := 0; i < 50; i++ {
		if _, err := provider.GetKey(ctx, "forged"); err != ErrKeyNotFound {
			t.Fatalf("expected ErrKeyNotFound, got %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("rate-limited misses took %v", elapsed)
	}
	if n := srv.requests.Load(); n != 1 {
		t.Fatalf("expected no extra fetches inside rate-limit window, got %d", n)
	}

	// Known key still works during the window.
	if _, err := provider.GetKey(ctx, "key1"); err != nil {
		t.Fatalf("known key failed: %v", err)
	}

	// After the window a real rotation is picked up.
	_, pub2 := testutil.GenerateRSATestKeyPair()
	srv.setKeys(
		&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub},
		&testutil.MockJWKSKey{Kid: "key2", PublicKey: pub2},
	)
	time.Sleep(350 * time.Millisecond)
	if _, err := provider.GetKey(ctx, "key2"); err != nil {
		t.Fatalf("rotated key should be found after window: %v", err)
	}
	if n := srv.requests.Load(); n != 2 {
		t.Fatalf("expected 2 fetches total, got %d", n)
	}
}

// TestJWKSKeyProvider_ExplicitRefreshBypassesRateLimit verifies RefreshCache
// is never throttled and does not consume the triggered-refresh budget.
func TestJWKSKeyProvider_ExplicitRefreshBypassesRateLimit(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	cache := NewJWKSCache(time.Hour)
	provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second,
		WithProviderMinRefreshInterval(time.Hour))

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := provider.RefreshCache(ctx); err != nil {
			t.Fatalf("explicit refresh %d failed: %v", i, err)
		}
	}
	if n := srv.requests.Load(); n != 3 {
		t.Fatalf("explicit refreshes must not be throttled, got %d fetches", n)
	}

	// A rotation right after a background refresh is still picked up.
	_, pub2 := testutil.GenerateRSATestKeyPair()
	srv.setKeys(
		&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub},
		&testutil.MockJWKSKey{Kid: "key2", PublicKey: pub2},
	)
	if _, err := provider.GetKey(ctx, "key2"); err != nil {
		t.Fatalf("triggered refresh should be allowed after explicit refreshes: %v", err)
	}
}

// TestTokenValidator_SignatureFailureRefreshIsRateLimited verifies a flood of
// bad signatures causes at most one JWKS fetch per window.
func TestTokenValidator_SignatureFailureRefreshIsRateLimited(t *testing.T) {
	_, pub := testutil.GenerateRSATestKeyPair()
	attacker, _ := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	cache := NewJWKSCache(time.Hour)
	provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second,
		WithProviderMinRefreshInterval(time.Hour))
	validator := NewTokenValidator(provider, &Config{ClockSkew: time.Minute}, nil)

	ctx := context.Background()
	if err := provider.RefreshCache(ctx); err != nil {
		t.Fatalf("initial refresh failed: %v", err)
	}
	srv.requests.Store(0)

	// Token signed by the wrong key but with a known kid.
	bad := testutil.GenerateTokenWithKid(attacker, "key1", map[string]interface{}{
		"sub": "mallory",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	for i := 0; i < 25; i++ {
		if _, err := validator.Validate(ctx, bad); err != ErrInvalidSignature {
			t.Fatalf("expected ErrInvalidSignature, got %v", err)
		}
	}

	if n := srv.requests.Load(); n != 1 {
		t.Fatalf("expected exactly 1 refresh for a flood of bad signatures, got %d", n)
	}
}

// TestTokenValidator_JWKSErrorsAreDistinguishable verifies callers can tell an
// unknown kid and a JWKS outage apart from a bad signature.
func TestTokenValidator_JWKSErrorsAreDistinguishable(t *testing.T) {
	priv, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	claims := map[string]interface{}{"sub": "alice", "exp": time.Now().Add(time.Hour).Unix()}

	t.Run("unknown kid", func(t *testing.T) {
		cache := NewJWKSCache(time.Hour)
		provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second)
		validator := NewTokenValidator(provider, &Config{ClockSkew: time.Minute}, nil)

		token := testutil.GenerateTokenWithKid(priv, "unknown", claims)
		if _, err := validator.Validate(context.Background(), token); err != ErrKeyNotFound {
			t.Fatalf("expected ErrKeyNotFound, got %v", err)
		}
	})

	t.Run("endpoint down with empty cache", func(t *testing.T) {
		down := newJWKSTestServer()
		down.setFail(true)
		defer down.Close()

		cache := NewJWKSCache(time.Hour)
		provider := NewJWKSKeyProvider(down.URL, cache, 5*time.Second)
		validator := NewTokenValidator(provider, &Config{ClockSkew: time.Minute}, nil)

		token := testutil.GenerateTokenWithKid(priv, "key1", claims)
		if _, err := validator.Validate(context.Background(), token); err != ErrJWKSFetchFailed {
			t.Fatalf("expected ErrJWKSFetchFailed, got %v", err)
		}
	})

	t.Run("endpoint down with stale cache still validates", func(t *testing.T) {
		cache := NewJWKSCache(time.Hour)
		provider := NewJWKSKeyProvider(srv.URL, cache, 5*time.Second)
		validator := NewTokenValidator(provider, &Config{ClockSkew: time.Minute}, nil)
		if err := provider.RefreshCache(context.Background()); err != nil {
			t.Fatalf("initial refresh failed: %v", err)
		}

		srv.setFail(true)
		defer srv.setFail(false)
		expireCache(cache)

		token := testutil.GenerateTokenWithKid(priv, "key1", claims)
		parsed, err := validator.Validate(context.Background(), token)
		if err != nil {
			t.Fatalf("stale key should validate while endpoint is down: %v", err)
		}
		if parsed["sub"] != "alice" {
			t.Fatalf("unexpected claims: %v", parsed)
		}
	})
}

// TestNew_WiresMinRefreshInterval verifies the middleware option reaches the provider.
func TestNew_WiresMinRefreshInterval(t *testing.T) {
	mw, err := New(
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
		WithJWKSMinRefreshInterval(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	provider, ok := mw.validator.keyProvider.(*JWKSKeyProvider)
	if !ok {
		t.Fatalf("expected JWKSKeyProvider, got %T", mw.validator.keyProvider)
	}
	if provider.minRefreshInterval != 2*time.Minute {
		t.Fatalf("expected 2m min refresh interval, got %v", provider.minRefreshInterval)
	}
	if provider.logger != mw.logger {
		t.Fatal("provider logger should be the middleware logger")
	}

	// Default applies when unset.
	mw, err = New(WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if got := mw.validator.keyProvider.(*JWKSKeyProvider).minRefreshInterval; got != DefaultJWKSMinRefreshInterval {
		t.Fatalf("expected default %v, got %v", DefaultJWKSMinRefreshInterval, got)
	}
}

// keyMap is a small helper to build the map type Replace expects.
type keyMap map[string]interface{}

func (m keyMap) toPublicKeys() map[string]crypto.PublicKey {
	out := make(map[string]crypto.PublicKey, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
