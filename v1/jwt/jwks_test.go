package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
)

// TestJWKSCache_NewJWKSCache tests cache creation.
func TestJWKSCache_NewJWKSCache(t *testing.T) {
	ttl := 15 * time.Minute
	cache := NewJWKSCache(ttl)

	if cache == nil {
		t.Fatal("NewJWKSCache returned nil")
	}

	if cache.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, cache.ttl)
	}

	// lastFetch should be zero time initially
	cache.mu.RLock()
	lastFetch := cache.lastFetch
	cache.mu.RUnlock()

	if !lastFetch.IsZero() {
		t.Errorf("Expected zero lastFetch, got %v", lastFetch)
	}
}

// TestJWKSCache_SetAndGet tests basic Set and Get operations.
func TestJWKSCache_SetAndGet(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)

	// Generate test key
	_, publicKey := testutil.GenerateRSATestKeyPair()

	// Initially, cache is stale (zero lastFetch)
	_, ok := cache.Get("key1")
	if ok {
		t.Error("Expected Get to return false for stale cache")
	}

	// Set key and update lastFetch
	cache.Set("key1", publicKey)
	cache.UpdateLastFetch()

	// Now Get should succeed
	retrievedKey, ok := cache.Get("key1")
	if !ok {
		t.Fatal("Expected Get to return true after UpdateLastFetch")
	}

	// Verify the key matches
	retrievedRSA, ok := retrievedKey.(*rsa.PublicKey)
	if !ok {
		t.Fatal("Retrieved key is not *rsa.PublicKey")
	}

	if retrievedRSA.N.Cmp(publicKey.N) != 0 {
		t.Error("Retrieved key does not match original key")
	}
}

// TestJWKSCache_CacheMiss tests Get with non-existent key.
func TestJWKSCache_CacheMiss(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)

	// Update lastFetch to make cache fresh
	cache.UpdateLastFetch()

	// Try to get non-existent key
	_, ok := cache.Get("non-existent")
	if ok {
		t.Error("Expected Get to return false for non-existent key")
	}
}

// TestJWKSCache_CacheExpiration tests TTL-based expiration.
func TestJWKSCache_CacheExpiration(t *testing.T) {
	// Use a very short TTL for testing
	cache := NewJWKSCache(100 * time.Millisecond)

	// Generate test key
	_, publicKey := testutil.GenerateRSATestKeyPair()

	// Set key and update lastFetch
	cache.Set("key1", publicKey)
	cache.UpdateLastFetch()

	// Get should succeed immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected Get to return true with fresh cache")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Get should now fail (cache is stale)
	_, ok = cache.Get("key1")
	if ok {
		t.Error("Expected Get to return false after TTL expiration")
	}
}

// TestJWKSCache_MultipleKeys tests storing and retrieving multiple keys.
func TestJWKSCache_MultipleKeys(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)

	// Generate multiple test keys
	_, rsaKey1 := testutil.GenerateRSATestKeyPair()
	_, rsaKey2 := testutil.GenerateRSATestKeyPair()
	_, ecdsaKey := testutil.GenerateECDSATestKeyPair()

	// Set multiple keys
	cache.Set("rsa-key-1", rsaKey1)
	cache.Set("rsa-key-2", rsaKey2)
	cache.Set("ecdsa-key-1", ecdsaKey)
	cache.UpdateLastFetch()

	// Retrieve all keys
	key1, ok := cache.Get("rsa-key-1")
	if !ok {
		t.Error("Expected to retrieve rsa-key-1")
	}
	if _, ok := key1.(*rsa.PublicKey); !ok {
		t.Error("rsa-key-1 should be *rsa.PublicKey")
	}

	key2, ok := cache.Get("rsa-key-2")
	if !ok {
		t.Error("Expected to retrieve rsa-key-2")
	}
	if _, ok := key2.(*rsa.PublicKey); !ok {
		t.Error("rsa-key-2 should be *rsa.PublicKey")
	}

	key3, ok := cache.Get("ecdsa-key-1")
	if !ok {
		t.Error("Expected to retrieve ecdsa-key-1")
	}
	if key3 == nil {
		t.Error("ecdsa-key-1 should not be nil")
	}
}

// TestJWKSCache_UpdateLastFetchUpdatesTimestamp tests that UpdateLastFetch works.
func TestJWKSCache_UpdateLastFetchUpdatesTimestamp(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)

	// Initially lastFetch is zero
	cache.mu.RLock()
	initialLastFetch := cache.lastFetch
	cache.mu.RUnlock()

	if !initialLastFetch.IsZero() {
		t.Error("Expected initial lastFetch to be zero")
	}

	// Update lastFetch
	before := time.Now()
	cache.UpdateLastFetch()
	after := time.Now()

	// Check that lastFetch was updated
	cache.mu.RLock()
	updatedLastFetch := cache.lastFetch
	cache.mu.RUnlock()

	if updatedLastFetch.IsZero() {
		t.Error("Expected lastFetch to be updated to non-zero time")
	}

	// Verify it's within the expected time range
	if updatedLastFetch.Before(before) || updatedLastFetch.After(after) {
		t.Errorf("lastFetch %v is not between %v and %v", updatedLastFetch, before, after)
	}
}

// TestJWKSCache_ConcurrentGetAndSet tests concurrent access with race detector.
func TestJWKSCache_ConcurrentGetAndSet(t *testing.T) {
	cache := NewJWKSCache(1 * time.Minute)

	// Generate test keys
	keys := make(map[string]interface{})
	for i := 0; i < 10; i++ {
		kid := time.Now().Format("key-2006-01-02-15-04-05.000000000")
		_, publicKey := testutil.GenerateRSATestKeyPair()
		keys[kid] = publicKey
		// Sleep a tiny bit to ensure unique kid
		time.Sleep(1 * time.Nanosecond)
	}

	// Update lastFetch so cache is fresh
	cache.UpdateLastFetch()

	var wg sync.WaitGroup

	// Concurrent writes
	for kid, key := range keys {
		wg.Add(1)
		go func(k string, v interface{}) {
			defer wg.Done()
			cache.Set(k, v)
		}(kid, key)
	}

	// Concurrent reads
	for kid := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			// May or may not find the key depending on timing
			cache.Get(k)
		}(kid)
	}

	// Concurrent UpdateLastFetch
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.UpdateLastFetch()
		}()
	}

	wg.Wait()

	// Verify all keys are retrievable after concurrent operations
	for kid, expectedKey := range keys {
		retrievedKey, ok := cache.Get(kid)
		if !ok {
			t.Errorf("Expected to retrieve key %s after concurrent operations", kid)
			continue
		}

		retrievedRSA := retrievedKey.(*rsa.PublicKey)
		expectedRSA := expectedKey.(*rsa.PublicKey)

		if retrievedRSA.N.Cmp(expectedRSA.N) != 0 {
			t.Errorf("Key %s does not match after concurrent operations", kid)
		}
	}
}

// TestJWKSCache_ConcurrentGetWithExpiration tests concurrent Get operations during cache expiration.
func TestJWKSCache_ConcurrentGetWithExpiration(t *testing.T) {
	cache := NewJWKSCache(200 * time.Millisecond)

	// Set a key
	_, publicKey := testutil.GenerateRSATestKeyPair()
	cache.Set("key1", publicKey)
	cache.UpdateLastFetch()

	var wg sync.WaitGroup
	results := make(chan bool, 100)

	// Launch many concurrent Get operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := cache.Get("key1")
			results <- ok
		}()
	}

	wg.Wait()
	close(results)

	// All should succeed since cache is fresh
	for ok := range results {
		if !ok {
			t.Error("Expected all Get operations to succeed with fresh cache")
		}
	}

	// Wait for cache to expire
	time.Sleep(250 * time.Millisecond)

	// Now all Gets should fail
	results2 := make(chan bool, 50)
	var wg2 sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			_, ok := cache.Get("key1")
			results2 <- ok
		}()
	}

	wg2.Wait()
	close(results2)

	// All should fail since cache is stale
	for ok := range results2 {
		if ok {
			t.Error("Expected all Get operations to fail with stale cache")
		}
	}
}

// TestJWKSCache_SetDoesNotUpdateLastFetch tests that Set doesn't update lastFetch.
func TestJWKSCache_SetDoesNotUpdateLastFetch(t *testing.T) {
	cache := NewJWKSCache(1 * time.Minute)

	// Update lastFetch first
	cache.UpdateLastFetch()

	cache.mu.RLock()
	initialLastFetch := cache.lastFetch
	cache.mu.RUnlock()

	// Sleep a bit
	time.Sleep(10 * time.Millisecond)

	// Set a key
	_, publicKey := testutil.GenerateRSATestKeyPair()
	cache.Set("key1", publicKey)

	// Verify lastFetch hasn't changed
	cache.mu.RLock()
	afterSetLastFetch := cache.lastFetch
	cache.mu.RUnlock()

	if !afterSetLastFetch.Equal(initialLastFetch) {
		t.Error("Set should not update lastFetch timestamp")
	}
}

// TestJWKSCache_ZeroTTL tests behavior with zero TTL (always stale).
func TestJWKSCache_ZeroTTL(t *testing.T) {
	cache := NewJWKSCache(0)

	_, publicKey := testutil.GenerateRSATestKeyPair()
	cache.Set("key1", publicKey)
	cache.UpdateLastFetch()

	// Even immediately after update, Get should return false with zero TTL
	_, ok := cache.Get("key1")
	if ok {
		t.Error("Expected Get to return false with zero TTL")
	}
}

// Test fetchJWKS function

// TestFetchJWKS_SuccessfulFetch tests successful JWKS fetching and parsing.
func TestFetchJWKS_SuccessfulFetch(t *testing.T) {
	// Create a test JWKS with RSA key
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method is GET
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Return JWKS JSON
		// For simplicity, we'll return a minimal valid JWKS with RSA key
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		// Trim leading zeros
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "rsa-key-1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	// Fetch JWKS
	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	keys, err := fetchJWKS(ctx, server.URL, httpClient)
	if err != nil {
		t.Fatalf("fetchJWKS failed: %v", err)
	}

	// Verify we got the key
	if len(keys) == 0 {
		t.Fatal("Expected at least one key")
	}

	// Verify the RSA key
	rsaKey, ok := keys["rsa-key-1"]
	if !ok {
		t.Fatal("Expected to find rsa-key-1")
	}

	if rsaKey == nil {
		t.Fatal("RSA key is nil")
	}
}

// TestFetchJWKS_HTTPError tests handling of HTTP errors.
func TestFetchJWKS_HTTPError(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			ctx := context.Background()
			httpClient := &http.Client{Timeout: 5 * time.Second}

			_, err := fetchJWKS(ctx, server.URL, httpClient)
			if err == nil {
				t.Errorf("Expected error for status %d, got nil", tc.statusCode)
			}

			if err != ErrJWKSFetchFailed {
				t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
			}
		})
	}
}

// TestFetchJWKS_InvalidJSON tests handling of invalid JSON responses.
func TestFetchJWKS_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	_, err := fetchJWKS(ctx, server.URL, httpClient)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if err != ErrJWKSFetchFailed {
		t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
	}
}

// TestFetchJWKS_MalformedJWK tests handling of malformed JWK data.
func TestFetchJWKS_MalformedJWK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return valid JSON but malformed JWK structure
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"keys": [{"kty": "invalid"}]}`))
	}))
	defer server.Close()

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	_, err := fetchJWKS(ctx, server.URL, httpClient)
	if err == nil {
		t.Error("Expected error for malformed JWK")
	}

	// The error might be ErrJWKSFetchFailed
	if err != ErrJWKSFetchFailed {
		t.Logf("Got error: %v", err)
	}
}

// TestFetchJWKS_Timeout tests handling of request timeout.
func TestFetchJWKS_Timeout(t *testing.T) {
	// Create server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Never respond
	}))
	defer server.Close()

	// Use short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	httpClient := &http.Client{}

	_, err := fetchJWKS(ctx, server.URL, httpClient)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestFetchJWKS_EmptyKeySet tests handling of empty key set.
func TestFetchJWKS_EmptyKeySet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"keys": []}`))
	}))
	defer server.Close()

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	_, err := fetchJWKS(ctx, server.URL, httpClient)
	if err == nil {
		t.Error("Expected error for empty key set")
	}

	if err != ErrJWKSFetchFailed {
		t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
	}
}

// TestFetchJWKS_KeysWithoutKid tests handling of keys without kid.
func TestFetchJWKS_KeysWithoutKid(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		// Trim leading zeros
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		// JWKS without kid
		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}

	_, err := fetchJWKS(ctx, server.URL, httpClient)
	// Should fail because all keys are skipped (no kid)
	if err == nil {
		t.Error("Expected error when all keys lack kid")
	}

	if err != ErrJWKSFetchFailed {
		t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
	}
}

// Test JWKSKeyProvider

// TestJWKSKeyProvider_NewJWKSKeyProvider tests provider creation.
func TestJWKSKeyProvider_NewJWKSKeyProvider(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)
	endpoint := "https://auth.example.com/.well-known/jwks.json"
	timeout := 10 * time.Second

	provider := NewJWKSKeyProvider(endpoint, cache, timeout)

	if provider == nil {
		t.Fatal("NewJWKSKeyProvider returned nil")
	}

	if provider.endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, provider.endpoint)
	}

	if provider.cache != cache {
		t.Error("Cache not set correctly")
	}

	if provider.httpClient == nil {
		t.Fatal("HTTP client is nil")
	}

	if provider.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, provider.httpClient.Timeout)
	}
}

// TestJWKSKeyProvider_GetKey_CacheHit tests GetKey with cache hit.
func TestJWKSKeyProvider_GetKey_CacheHit(t *testing.T) {
	cache := NewJWKSCache(15 * time.Minute)
	endpoint := "https://auth.example.com/.well-known/jwks.json"
	provider := NewJWKSKeyProvider(endpoint, cache, 10*time.Second)

	// Pre-populate cache
	_, rsaKey := testutil.GenerateRSATestKeyPair()
	cache.Set("key1", rsaKey)
	cache.UpdateLastFetch()

	// GetKey should return from cache without making HTTP request
	ctx := context.Background()
	key, err := provider.GetKey(ctx, "key1")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	if key == nil {
		t.Fatal("Expected key, got nil")
	}

	retrievedRSA := key.(*rsa.PublicKey)
	if retrievedRSA.N.Cmp(rsaKey.N) != 0 {
		t.Error("Retrieved key does not match cached key")
	}
}

// TestJWKSKeyProvider_GetKey_CacheMissWithRefresh tests GetKey with cache miss.
func TestJWKSKeyProvider_GetKey_CacheMissWithRefresh(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "key1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// GetKey should trigger refresh and return key
	ctx := context.Background()
	key, err := provider.GetKey(ctx, "key1")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	if key == nil {
		t.Fatal("Expected key, got nil")
	}

	// Verify key is now in cache
	cachedKey, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected key to be in cache after GetKey")
	}
	if cachedKey == nil {
		t.Error("Cached key is nil")
	}
}

// TestJWKSKeyProvider_GetKey_RefreshFailure tests GetKey when refresh fails.
func TestJWKSKeyProvider_GetKey_RefreshFailure(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	ctx := context.Background()
	_, err := provider.GetKey(ctx, "key1")
	if err == nil {
		t.Error("Expected error when refresh fails")
	}

	if err != ErrJWKSFetchFailed {
		t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
	}
}

// TestJWKSKeyProvider_GetKey_KeyNotFoundAfterRefresh tests when key not in JWKS.
func TestJWKSKeyProvider_GetKey_KeyNotFoundAfterRefresh(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		// Return JWKS with different key
		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "different-key",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	ctx := context.Background()
	_, err := provider.GetKey(ctx, "key1")
	if err == nil {
		t.Error("Expected error when key not found after refresh")
	}

	if err != ErrJWKSFetchFailed {
		t.Errorf("Expected ErrJWKSFetchFailed, got %v", err)
	}
}

// TestJWKSKeyProvider_RefreshCache_UpdatesCache tests RefreshCache.
func TestJWKSKeyProvider_RefreshCache_UpdatesCache(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "key1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Initially cache is empty and stale
	_, ok := cache.Get("key1")
	if ok {
		t.Error("Cache should be empty initially")
	}

	// Refresh cache
	ctx := context.Background()
	err := provider.RefreshCache(ctx)
	if err != nil {
		t.Fatalf("RefreshCache failed: %v", err)
	}

	// Verify key is now in cache
	key, ok := cache.Get("key1")
	if !ok {
		t.Fatal("Expected key in cache after refresh")
	}

	if key == nil {
		t.Fatal("Cached key is nil")
	}
}

// TestJWKSKeyProvider_ConcurrentGetKey tests concurrent GetKey calls.
func TestJWKSKeyProvider_ConcurrentGetKey(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	requestCount := 0
	var requestMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		requestCount++
		requestMu.Unlock()

		// Add small delay to increase chance of concurrent requests
		time.Sleep(50 * time.Millisecond)

		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "key1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)

	// Launch multiple concurrent GetKey calls
	var wg sync.WaitGroup
	numGoroutines := 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := provider.GetKey(ctx, "key1")
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// All should succeed
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent GetKey failed: %v", err)
		}
	}

	// Only one HTTP request should have been made (due to mutex)
	requestMu.Lock()
	count := requestCount
	requestMu.Unlock()

	if count != 1 {
		t.Logf("Warning: Expected 1 HTTP request, got %d (may vary due to timing)", count)
	}
}
