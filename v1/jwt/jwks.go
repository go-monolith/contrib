package jwt

import (
	"context"
	"crypto"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// JWKSCache caches JWKS keys with TTL-based expiration.
//
// Thread-safe for concurrent access.
type JWKSCache struct {
	keys      sync.Map  // kid -> crypto.PublicKey
	lastFetch time.Time
	ttl       time.Duration
	mu        sync.RWMutex // for lastFetch access
}

// NewJWKSCache creates a new JWKS cache with the given TTL.
//
// The cache is initially empty with a zero lastFetch time, which means
// the first Get() call will indicate the cache is stale and needs refresh.
//
// Parameters:
//   - ttl: Time-to-live duration for the cache. After this duration,
//     Get() will return false to trigger a cache refresh.
//
// Example:
//
//	cache := NewJWKSCache(15 * time.Minute)
func NewJWKSCache(ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		ttl: ttl,
		// lastFetch is zero time initially, so first Get() will trigger a fetch
	}
}

// Get retrieves a key from the cache by kid.
//
// This method checks if the cache is stale before returning the key.
// If the cache is stale (time.Since(lastFetch) > ttl), it returns (nil, false)
// to signal that a refresh is needed.
//
// Parameters:
//   - kid: The key ID to look up
//
// Returns:
//   - The public key (crypto.PublicKey) if found and cache is fresh
//   - A boolean indicating if the key was found and cache is fresh
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	key, ok := cache.Get("key-id-123")
//	if !ok {
//	    // Cache miss or stale - need to refresh
//	    refreshCache()
//	    key, ok = cache.Get("key-id-123")
//	}
func (c *JWKSCache) Get(kid string) (interface{}, bool) {
	// Check if cache is stale
	c.mu.RLock()
	lastFetch := c.lastFetch
	c.mu.RUnlock()

	// If cache is stale, return false to trigger refresh
	if time.Since(lastFetch) > c.ttl {
		return nil, false
	}

	// Load key from sync.Map
	value, ok := c.keys.Load(kid)
	if !ok {
		return nil, false
	}

	return value, true
}

// Set stores a key in the cache with the given kid.
//
// This method does NOT update the lastFetch timestamp. Use UpdateLastFetch()
// after setting all keys from a JWKS fetch operation.
//
// Parameters:
//   - kid: The key ID
//   - key: The public key (crypto.PublicKey)
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	cache.Set("key-id-123", rsaPublicKey)
//	cache.Set("key-id-456", ecdsaPublicKey)
//	cache.UpdateLastFetch() // Mark cache as fresh
func (c *JWKSCache) Set(kid string, key interface{}) {
	c.keys.Store(kid, key)
}

// UpdateLastFetch updates the last fetch timestamp to the current time.
//
// This method should be called after successfully fetching and storing
// all keys from a JWKS endpoint to mark the cache as fresh.
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	// Fetch keys from JWKS endpoint
//	keys, err := fetchJWKS(ctx, endpoint, httpClient)
//	if err != nil {
//	    return err
//	}
//
//	// Update cache with all keys
//	for kid, key := range keys {
//	    cache.Set(kid, key)
//	}
//
//	// Mark cache as fresh
//	cache.UpdateLastFetch()
func (c *JWKSCache) UpdateLastFetch() {
	c.mu.Lock()
	c.lastFetch = time.Now()
	c.mu.Unlock()
}

// Clear removes all cached keys.
//
// This method is typically called before refreshing the cache to ensure
// that retired keys are removed and don't persist in the cache.
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	cache.Clear()
//	for kid, key := range newKeys {
//	    cache.Set(kid, key)
//	}
//	cache.UpdateLastFetch()
func (c *JWKSCache) Clear() {
	// Create a new sync.Map to effectively clear all entries
	c.keys = sync.Map{}
}

// JWKSKeyProvider provides keys by fetching from a JWKS endpoint.
type JWKSKeyProvider struct {
	endpoint   string
	cache      *JWKSCache
	httpClient *http.Client
	refreshMu  sync.Mutex // prevents concurrent JWKS refreshes
}

// NewJWKSKeyProvider creates a new JWKS key provider.
//
// The provider fetches public keys from a JWKS endpoint and caches them.
// It implements the KeyProvider interface for use with TokenValidator.
//
// Parameters:
//   - endpoint: The JWKS endpoint URL (e.g., "https://auth.example.com/.well-known/jwks.json")
//   - cache: The cache to use for storing fetched keys
//   - timeout: HTTP request timeout for JWKS fetches
//
// Returns a new JWKSKeyProvider instance.
//
// Example:
//
//	cache := NewJWKSCache(15 * time.Minute)
//	provider := NewJWKSKeyProvider("https://auth.example.com/.well-known/jwks.json", cache, 10*time.Second)
func NewJWKSKeyProvider(endpoint string, cache *JWKSCache, timeout time.Duration) *JWKSKeyProvider {
	return &JWKSKeyProvider{
		endpoint: endpoint,
		cache:    cache,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetKey retrieves a key by kid, fetching from JWKS if not cached.
//
// This method implements the KeyProvider interface. It follows this flow:
//  1. Try to get the key from cache
//  2. If cache hit, return the key
//  3. If cache miss, refresh the cache from JWKS endpoint
//  4. Retry cache lookup after refresh
//  5. Return error if key still not found
//
// Parameters:
//   - ctx: Context for request cancellation
//   - kid: The key ID to retrieve
//
// Returns:
//   - The public key (crypto.PublicKey) if found
//   - An error if the key cannot be retrieved
//
// Thread-safe: Multiple goroutines can call this concurrently.
// Concurrent refresh attempts are deduplicated using refreshMu.
//
// Example:
//
//	key, err := provider.GetKey(ctx, "key-id-123")
//	if err != nil {
//	    return fmt.Errorf("failed to get key: %w", err)
//	}
func (p *JWKSKeyProvider) GetKey(ctx context.Context, kid string) (interface{}, error) {
	// Try to get from cache first
	key, ok := p.cache.Get(kid)
	if ok {
		return key, nil
	}

	// Cache miss - refresh cache
	if err := p.RefreshCache(ctx); err != nil {
		return nil, err
	}

	// Retry cache lookup after refresh
	key, ok = p.cache.Get(kid)
	if !ok {
		// Key still not found after refresh
		return nil, ErrJWKSFetchFailed
	}

	return key, nil
}

// RefreshCache fetches the JWKS and updates the cache.
//
// This method fetches keys from the JWKS endpoint and updates the cache.
// It uses a mutex to prevent concurrent refresh attempts, ensuring only
// one refresh happens at a time.
//
// Parameters:
//   - ctx: Context for request cancellation
//
// Returns an error if the JWKS fetch fails.
//
// Thread-safe: Multiple goroutines calling this will be serialized.
// Only the first caller will actually fetch; others will wait.
//
// Example:
//
//	if err := provider.RefreshCache(ctx); err != nil {
//	    log.Printf("Failed to refresh JWKS cache: %v", err)
//	}
func (p *JWKSKeyProvider) RefreshCache(ctx context.Context) error {
	// Use mutex to prevent concurrent refreshes
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()

	// Fetch JWKS from endpoint
	keys, err := fetchJWKS(ctx, p.endpoint, p.httpClient)
	if err != nil {
		return err
	}

	// Clear old keys before adding new ones
	// This ensures retired keys are removed from the cache
	p.cache.Clear()

	// Update cache with all fetched keys
	for kid, key := range keys {
		p.cache.Set(kid, key)
	}

	// Mark cache as fresh
	p.cache.UpdateLastFetch()

	return nil
}

// fetchJWKS fetches the JWKS document from the endpoint and parses it.
//
// This function:
//  1. Makes an HTTP GET request to the JWKS endpoint
//  2. Parses the JSON response as a JWK Set
//  3. Converts each JWK to a crypto.PublicKey
//  4. Returns a map of kid (key ID) to public key
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - endpoint: The JWKS endpoint URL (e.g., "https://auth.example.com/.well-known/jwks.json")
//   - httpClient: HTTP client to use for the request
//
// Returns:
//   - A map of kid → crypto.PublicKey
//   - An error if the fetch or parsing fails
//
// Error cases:
//   - Network errors (connection failed, timeout)
//   - Non-200 HTTP status codes
//   - Invalid JSON response
//   - Malformed JWK data
//
// Example:
//
//	httpClient := &http.Client{Timeout: 10 * time.Second}
//	keys, err := fetchJWKS(ctx, "https://auth.example.com/.well-known/jwks.json", httpClient)
//	if err != nil {
//	    return fmt.Errorf("failed to fetch JWKS: %w", err)
//	}
func fetchJWKS(ctx context.Context, endpoint string, httpClient *http.Client) (map[string]crypto.PublicKey, error) {
	// Create HTTP GET request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Set Accept header for JSON
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, ErrJWKSFetchFailed
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, ErrJWKSFetchFailed
	}

	// Parse JWKS using jwx library
	// The jwx library handles all the JWK parsing details
	set, err := jwk.ParseReader(resp.Body)
	if err != nil {
		return nil, ErrJWKSFetchFailed
	}

	// Convert JWK Set to map of kid → crypto.PublicKey
	result := make(map[string]crypto.PublicKey)

	// Iterate through all keys in the set
	iter := set.Keys(ctx)
	for iter.Next(ctx) {
		pair := iter.Pair()
		key := pair.Value.(jwk.Key)

		// Get the key ID (kid)
		kid := key.KeyID()
		if kid == "" {
			// Skip keys without kid - we can't cache them
			continue
		}

		// Convert JWK to raw crypto.PublicKey
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			// Skip keys that can't be converted
			continue
		}

		// Verify it's a public key type we support
		publicKey, ok := rawKey.(crypto.PublicKey)
		if !ok {
			// Skip non-public keys
			continue
		}

		result[kid] = publicKey
	}

	if len(result) == 0 {
		return nil, ErrJWKSFetchFailed
	}

	return result, nil
}
