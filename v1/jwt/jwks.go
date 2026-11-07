package jwt

import (
	"context"
	"crypto"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// DefaultJWKSMinRefreshInterval is the default minimum interval between two
// JWKS refreshes that are triggered by token validation (unknown kid, stale
// cache, or signature failure). Background and explicit refreshes are not
// rate-limited.
const DefaultJWKSMinRefreshInterval = 30 * time.Second

// JWKSCache caches JWKS keys with TTL-based expiration.
//
// Keys are kept in the cache after the TTL expires. A lookup then reports
// the key as stale so that the caller can serve it while a refresh runs in
// the background (stale-while-revalidate). Keys only disappear from the
// cache when a refresh replaces the key set or Clear() is called.
//
// Thread-safe for concurrent access.
type JWKSCache struct {
	keys      map[string]interface{} // kid -> crypto.PublicKey
	lastFetch time.Time
	ttl       time.Duration
	mu        sync.RWMutex // guards keys and lastFetch
}

// NewJWKSCache creates a new JWKS cache with the given TTL.
//
// The cache is initially empty with a zero lastFetch time, which means
// the first Get() call will indicate the cache is stale and needs refresh.
//
// Parameters:
//   - ttl: Time-to-live duration for the cache. After this duration,
//     Get() will return false and Lookup() reports the keys as stale.
//
// Example:
//
//	cache := NewJWKSCache(15 * time.Minute)
func NewJWKSCache(ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		keys: make(map[string]interface{}),
		ttl:  ttl,
		// lastFetch is zero time initially, so first Get() will trigger a fetch
	}
}

// Get retrieves a key from the cache by kid, but only while the cache is fresh.
//
// If the cache is stale (time.Since(lastFetch) > ttl) or the key is unknown,
// it returns (nil, false). Use Lookup() to read stale keys.
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
	key, found, fresh := c.Lookup(kid)
	if !found || !fresh {
		return nil, false
	}
	return key, true
}

// Lookup retrieves a key from the cache by kid regardless of freshness.
//
// Returns:
//   - key: the cached key, or nil when not found
//   - found: whether the kid is in the cache
//   - fresh: whether the cache is within its TTL
//
// A found but not fresh key may still be served while a refresh runs in the
// background. Both flags are read under the same lock, so they are consistent
// with each other.
//
// Thread-safe: Safe for concurrent access.
func (c *JWKSCache) Lookup(kid string) (key interface{}, found bool, fresh bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, found = c.keys[kid]
	fresh = time.Since(c.lastFetch) <= c.ttl
	return key, found, fresh
}

// IsFresh reports whether the cache is within its TTL.
//
// Thread-safe: Safe for concurrent access.
func (c *JWKSCache) IsFresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastFetch) <= c.ttl
}

// Set stores a key in the cache with the given kid.
//
// This method does NOT update the lastFetch timestamp. Use UpdateLastFetch()
// after setting all keys from a JWKS fetch operation. Prefer Replace() to
// swap in a whole key set atomically.
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	cache.Set("key-id-123", rsaPublicKey)
//	cache.Set("key-id-456", ecdsaPublicKey)
//	cache.UpdateLastFetch() // Mark cache as fresh
func (c *JWKSCache) Set(kid string, key interface{}) {
	c.mu.Lock()
	c.keys[kid] = key
	c.mu.Unlock()
}

// Replace atomically swaps the whole key set and marks the cache as fresh.
//
// Readers never observe an empty cache during a refresh, unlike a
// Clear()/Set()/UpdateLastFetch() sequence. Retired keys that are absent
// from the new set are dropped.
//
// Thread-safe: Safe for concurrent access.
//
// Example:
//
//	keys, err := fetchJWKS(ctx, endpoint, httpClient)
//	if err != nil {
//	    return err
//	}
//	cache.Replace(keys)
func (c *JWKSCache) Replace(keys map[string]crypto.PublicKey) {
	newKeys := make(map[string]interface{}, len(keys))
	for kid, key := range keys {
		newKeys[kid] = key
	}

	c.mu.Lock()
	c.keys = newKeys
	c.lastFetch = time.Now()
	c.mu.Unlock()
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
//	for kid, key := range keys {
//	    cache.Set(kid, key)
//	}
//	cache.UpdateLastFetch()
func (c *JWKSCache) UpdateLastFetch() {
	c.mu.Lock()
	c.lastFetch = time.Now()
	c.mu.Unlock()
}

// Clear removes all cached keys.
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
	c.mu.Lock()
	c.keys = make(map[string]interface{})
	c.mu.Unlock()
}

// JWKSProviderOption configures a JWKSKeyProvider.
type JWKSProviderOption func(*JWKSKeyProvider)

// WithProviderMinRefreshInterval sets the minimum interval between two
// refreshes triggered by token validation (unknown kid, stale cache, or
// signature failure). Set to 0 to disable rate limiting.
//
// Default: DefaultJWKSMinRefreshInterval (30 seconds)
func WithProviderMinRefreshInterval(interval time.Duration) JWKSProviderOption {
	return func(p *JWKSKeyProvider) {
		p.minRefreshInterval = interval
	}
}

// WithProviderLogger sets the logger used to report background refresh
// failures and skipped refreshes.
//
// Default: slog.Default()
func WithProviderLogger(logger *slog.Logger) JWKSProviderOption {
	return func(p *JWKSKeyProvider) {
		if logger != nil {
			p.logger = logger
		}
	}
}

// JWKSKeyProvider provides keys by fetching from a JWKS endpoint.
//
// Behaviour under load or when the JWKS endpoint is slow:
//   - A known key whose cache entry is past its TTL is served immediately
//     (stale-while-revalidate). A single background refresh is started;
//     the caller never waits for the network.
//   - An unknown kid triggers a synchronous refresh, but at most once per
//     minRefreshInterval. Within that window the key is reported as not
//     found without touching the network.
//   - Explicit RefreshCache() calls (startup, background ticker) are never
//     rate-limited.
type JWKSKeyProvider struct {
	endpoint   string
	cache      *JWKSCache
	httpClient *http.Client
	logger     *slog.Logger

	refreshMu sync.Mutex // serializes JWKS fetches

	minRefreshInterval time.Duration
	attemptMu          sync.Mutex // guards lastAttempt
	lastAttempt        time.Time  // last *triggered* refresh attempt (success or failure)

	backgroundRefreshing atomic.Bool // single-flight flag for stale-triggered refreshes
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
//   - opts: Optional provider settings (rate limit, logger)
//
// Returns a new JWKSKeyProvider instance.
//
// Example:
//
//	cache := NewJWKSCache(15 * time.Minute)
//	provider := NewJWKSKeyProvider("https://auth.example.com/.well-known/jwks.json", cache, 10*time.Second,
//	    WithProviderMinRefreshInterval(time.Minute),
//	)
func NewJWKSKeyProvider(endpoint string, cache *JWKSCache, timeout time.Duration, opts ...JWKSProviderOption) *JWKSKeyProvider {
	p := &JWKSKeyProvider{
		endpoint: endpoint,
		cache:    cache,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger:             slog.Default(),
		minRefreshInterval: DefaultJWKSMinRefreshInterval,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// GetKey retrieves a key by kid, fetching from JWKS if not cached.
//
// This method implements the KeyProvider interface. It follows this flow:
//  1. Known key, cache fresh: return it.
//  2. Known key, cache stale: return it and start one background refresh.
//  3. Unknown key: run a rate-limited synchronous refresh, then look again.
//
// Parameters:
//   - ctx: Context for request cancellation
//   - kid: The key ID to retrieve
//
// Returns:
//   - The public key (crypto.PublicKey) if found
//   - ErrJWKSFetchFailed if a refresh was attempted and failed
//   - ErrKeyNotFound if the kid is not in the key set (or the refresh was rate-limited)
//
// Thread-safe: Multiple goroutines can call this concurrently.
//
// Example:
//
//	key, err := provider.GetKey(ctx, "key-id-123")
//	if err != nil {
//	    return fmt.Errorf("failed to get key: %w", err)
//	}
func (p *JWKSKeyProvider) GetKey(ctx context.Context, kid string) (interface{}, error) {
	key, found, fresh := p.cache.Lookup(kid)
	if found {
		if !fresh {
			// Serve the stale key and revalidate in the background.
			p.triggerBackgroundRefresh()
		}
		return key, nil
	}

	// Unknown kid: refresh synchronously, subject to the rate limit.
	if _, err := p.refreshIfAllowed(ctx); err != nil {
		return nil, err
	}

	// Retry cache lookup after refresh. A stale hit is acceptable here: it
	// means the refresh was rate-limited and the key was added concurrently.
	key, found, _ = p.cache.Lookup(kid)
	if !found {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// RefreshCache fetches the JWKS and replaces the cache contents.
//
// This call is never rate-limited. It is used at startup and by the
// background ticker. Concurrent callers are serialized; the cache is swapped
// atomically so readers never see an empty key set.
//
// Parameters:
//   - ctx: Context for request cancellation
//
// Returns an error if the JWKS fetch fails. The previous keys stay in the
// cache on failure.
//
// Thread-safe: Multiple goroutines calling this will be serialized.
//
// Example:
//
//	if err := provider.RefreshCache(ctx); err != nil {
//	    log.Printf("Failed to refresh JWKS cache: %v", err)
//	}
func (p *JWKSKeyProvider) RefreshCache(ctx context.Context) error {
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()

	return p.refreshLocked(ctx)
}

// refreshLocked performs the fetch and cache swap. Caller must hold refreshMu.
func (p *JWKSKeyProvider) refreshLocked(ctx context.Context) error {
	keys, err := fetchJWKS(ctx, p.endpoint, p.httpClient)
	if err != nil {
		return err
	}

	p.cache.Replace(keys)
	return nil
}

// refreshIfAllowed runs a synchronous refresh unless one was attempted less
// than minRefreshInterval ago.
//
// Returns attempted=false (and a nil error) when the refresh was skipped.
// Concurrent callers wait on the refresh in flight; once it finishes they
// observe the fresh lastAttempt and skip their own fetch.
func (p *JWKSKeyProvider) refreshIfAllowed(ctx context.Context) (attempted bool, err error) {
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()

	if !p.refreshAllowed() {
		p.logger.Debug("JWKS refresh skipped: rate limited",
			"min_interval", p.minRefreshInterval,
		)
		return false, nil
	}

	// Only triggered refreshes count toward the rate limit, so a key
	// rotation right after a background refresh is still picked up at once.
	p.markAttempt()

	return true, p.refreshLocked(ctx)
}

// refreshAllowed reports whether enough time has passed since the last attempt.
func (p *JWKSKeyProvider) refreshAllowed() bool {
	if p.minRefreshInterval <= 0 {
		return true
	}
	p.attemptMu.Lock()
	defer p.attemptMu.Unlock()
	return time.Since(p.lastAttempt) >= p.minRefreshInterval
}

// markAttempt records the time of a refresh attempt.
func (p *JWKSKeyProvider) markAttempt() {
	p.attemptMu.Lock()
	p.lastAttempt = time.Now()
	p.attemptMu.Unlock()
}

// triggerBackgroundRefresh starts at most one background refresh at a time,
// subject to the rate limit. It returns immediately.
//
// The refresh runs with its own context bounded by the HTTP client timeout,
// so it does not depend on the caller's context and cannot leak.
func (p *JWKSKeyProvider) triggerBackgroundRefresh() {
	if !p.refreshAllowed() {
		return
	}
	if !p.backgroundRefreshing.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer p.backgroundRefreshing.Store(false)

		timeout := p.httpClient.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Second)
		defer cancel()

		attempted, err := p.refreshIfAllowed(ctx)
		if !attempted {
			return
		}
		if err != nil {
			p.logger.Warn("background JWKS refresh failed; serving stale keys",
				"endpoint", p.endpoint,
				"error", err,
			)
			return
		}
		p.logger.Debug("background JWKS refresh after stale hit succeeded")
	}()
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
	defer func() {
		_ = resp.Body.Close() //nolint:errcheck // Error on close is not actionable in defer
	}()

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
		key, ok := pair.Value.(jwk.Key)
		if !ok {
			// Skip invalid key types
			continue
		}

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
