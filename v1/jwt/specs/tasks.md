# JWT Middleware Implementation Plan (Mono Framework)

## Milestones

1. **Milestone 1 - Core Infrastructure & Configuration**
   - Tasks: 1-6
   - Notes: Set up project structure, configuration, and basic types

2. **Milestone 2 - Token Validation (Static Key Mode)**
   - Tasks: 7-13
   - Notes: Implement JWT validation with static secret/public key

3. **Milestone 3 - JWKS Support**
   - Tasks: 14-19
   - Notes: Implement JWKS fetching, caching, and background refresh

4. **Milestone 4 - Mono Middleware Integration**
   - Tasks: 20-26
   - Notes: Implement MiddlewareModule interface and handler wrapping

5. **Milestone 5 - Testing & Documentation**
   - Tasks: 27-33
   - Notes: Comprehensive tests, examples, and documentation

## Task List

### Milestone 1: Core Infrastructure & Configuration

- [x] 1. Initialize Go module and project structure
  - Navigate to: `~/Projects/myspec/mono_framework/contrib/v1/jwt/`
  - Note: `REQUIREMENTS.md` already exists - keep it for reference
  - Initialize Go module: `go mod init github.com/go-monolith/contrib/v1/jwt`
  - Install dependencies:
    - `go get github.com/go-monolith/mono@latest`
    - `go get github.com/golang-jwt/jwt/v5@latest`
    - `go get github.com/lestrrat-go/jwx/v2@latest` (for JWKS parsing)
  - Create package files:
    - `jwt.go` (main middleware implementation)
    - `config.go` (configuration struct and options)
    - `options.go` (functional options)
    - `validator.go` (JWT validation logic)
    - `provider.go` (key provider interfaces and implementations)
    - `jwks.go` (JWKS fetcher and cache)
    - `context.go` (context keys and helpers)
    - `errors.go` (custom error types)
  - _Requirements: NFR1_

- [x] 2. Define error types and constants
  - Create `errors.go` with error definitions:
    - `ErrMissingAuthHeader = errors.New("missing authorization header")`
    - `ErrInvalidAuthHeader = errors.New("invalid authorization header format")`
    - `ErrInvalidToken = errors.New("invalid token")`
    - `ErrTokenExpired = errors.New("token expired")`
    - `ErrTokenNotYetValid = errors.New("token not yet valid")`
    - `ErrInvalidSignature = errors.New("invalid token signature")`
    - `ErrInvalidIssuer = errors.New("invalid issuer")`
    - `ErrInvalidAudience = errors.New("invalid audience")`
    - `ErrInvalidIssuedAt = errors.New("invalid issued at time")`
    - `ErrInvalidClaims = errors.New("invalid claims format")`
    - `ErrMissingRequiredClaim = errors.New("missing required claim")`
    - `ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")`
  - Add godoc comments for each error
  - _Requirements: FR2, FR3, FR4, FR5_

- [x] 3. Define configuration struct
  - Create `config.go` with `Config` struct:
    - Key sources: `Secret []byte`, `PublicKey crypto.PublicKey`, `JWKSEndpoint string`
    - JWKS settings: `JWKSCacheTTL`, `JWKSRefreshInterval`, `JWKSRequestTimeout time.Duration`
    - Validation: `RequiredClaims []string`, `ExpectedIssuer string`, `ExpectedAudience []string`, `AllowedAlgorithms []string`, `ClockSkew time.Duration`
    - Header settings: `HeaderKey string`, `TokenPrefix string`
    - Behavior: `SkipPaths []string`, `Optional bool`
  - Implement `validateConfig(cfg *Config) error`:
    - Ensure exactly ONE key source is configured (mutually exclusive)
    - Validate JWKS settings if endpoint is configured
    - Return descriptive errors for invalid config
  - _Requirements: FR11_

  - [x] 3.1 Create configuration validation unit tests
    - Test valid configurations (secret, public key, JWKS)
    - Test error when no key source is provided
    - Test error when multiple key sources are provided
    - Test default value application
    - _Requirements: FR11_

- [x] 4. Implement functional options pattern
  - Create `options.go` with `Option` type: `type Option func(*Config)`
  - Implement option functions:
    - `WithSecret(secret []byte) Option`
    - `WithPublicKey(publicKey crypto.PublicKey) Option`
    - `WithJWKSEndpoint(endpoint string) Option`
    - `WithJWKSCacheTTL(ttl time.Duration) Option`
    - `WithJWKSRefreshInterval(interval time.Duration) Option`
    - `WithExpectedIssuer(issuer string) Option`
    - `WithExpectedAudience(audience ...string) Option`
    - `WithRequiredClaims(claims ...string) Option`
    - `WithAllowedAlgorithms(algorithms ...string) Option`
    - `WithClockSkew(skew time.Duration) Option`
    - `WithSkipPaths(paths ...string) Option`
    - `WithOptional(optional bool) Option`
  - Add godoc comments with usage examples
  - _Requirements: FR11_

- [x] 5. Implement context keys and helpers
  - Create `context.go` with unexported context key:
    ```go
    type claimsContextKey struct{}
    ```
  - Implement `WithClaims(ctx context.Context, claims jwt.MapClaims) context.Context`
  - Implement `ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool)`
  - Implement `MustClaimsFromContext(ctx context.Context) jwt.MapClaims` (panics if not found)
  - Implement `SubjectFromContext(ctx context.Context) (string, bool)` helper
  - Implement `IssuerFromContext(ctx context.Context) (string, bool)` helper
  - Add godoc comments with usage examples
  - _Requirements: FR6_

  - [x] 5.1 Create context helpers unit tests
    - Test WithClaims and ClaimsFromContext round-trip
    - Test MustClaimsFromContext panic behavior
    - Test SubjectFromContext and IssuerFromContext
    - Test handling of missing claims
    - _Requirements: FR6_

- [x] 6. Implement PEM key parsing helpers
  - Create helper functions in `config.go` or `helpers.go`:
    - `ParseRSAPublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error)`
    - `ParseECDSAPublicKeyFromPEM(pemData []byte) (*ecdsa.PublicKey, error)`
  - Support both PKCS#1 and PKIX formats
  - Return descriptive errors for invalid PEM data
  - _Requirements: FR11 (ease of configuration)_

  - [x] 6.1 Create PEM parsing unit tests
    - Test valid RSA PEM parsing
    - Test valid ECDSA PEM parsing
    - Test invalid PEM format handling
    - Test both PKCS#1 and PKIX formats
    - _Requirements: FR11_

### Milestone 2: Token Validation (Static Key Mode)

- [x] 7. Implement token extractor
  - Create `extractToken(headers map[string][]string, headerKey, tokenPrefix string) (string, error)` function
  - Perform case-insensitive lookup for header key
  - Validate header format: "<prefix> <token>"
  - Support case-insensitive prefix matching
  - Return `ErrMissingAuthHeader` if not found
  - Return `ErrInvalidAuthHeader` if format is invalid
  - Return extracted token string
  - _Requirements: FR1_

  - [x] 7.1 Create token extractor unit tests
    - Test valid header extraction
    - Test case-insensitive header key
    - Test case-insensitive "Bearer" prefix
    - Test missing header
    - Test invalid format (no prefix, wrong prefix, empty token)
    - _Requirements: FR1_

- [x] 8. Implement KeyProvider interface and static provider
  - Create `provider.go` with `KeyProvider` interface:
    ```go
    type KeyProvider interface {
        GetKey(ctx context.Context, kid string) (interface{}, error)
    }
    ```
  - Create `StaticKeyProvider` struct:
    - `key interface{}` (can be `[]byte` for HMAC or `crypto.PublicKey` for RSA/ECDSA)
  - Implement `NewStaticKeyProvider(key interface{}) *StaticKeyProvider`
  - Implement `GetKey(ctx context.Context, kid string) (interface{}, error)`:
    - Return `key` directly (ignore `kid` parameter)
  - _Requirements: FR2_

  - [x] 8.1 Create static key provider unit tests
    - Test GetKey returns correct key for HMAC secret
    - Test GetKey returns correct key for RSA public key
    - Test GetKey returns correct key for ECDSA public key
    - Test kid parameter is ignored
    - _Requirements: FR2_

- [x] 9. Implement token validator
  - Create `validator.go` with `TokenValidator` struct:
    - `keyProvider KeyProvider`
    - `config *Config`
  - Implement `NewTokenValidator(keyProvider KeyProvider, config *Config) *TokenValidator`
  - Implement `Validate(ctx context.Context, tokenString string) (jwt.MapClaims, error)`:
    - Parse JWT using `jwt.Parse()` with custom key function
    - Verify algorithm is in `AllowedAlgorithms` (if configured)
    - Get key from `keyProvider.GetKey(ctx, kid)`
    - Return parsed token claims if valid
  - _Requirements: FR2_

  - [x] 9.1 Create token validator unit tests
    - Test valid token parsing with HMAC
    - Test valid token parsing with RSA
    - Test valid token parsing with ECDSA
    - Test invalid signature rejection
    - Test unsupported algorithm rejection
    - Test malformed token rejection
    - Mock KeyProvider for controlled testing
    - _Requirements: FR2_

- [x] 10. Implement standard claims validation
  - Add methods to `TokenValidator`:
    - `validateStandardClaims(claims jwt.MapClaims) error`
    - `validateIssuer(claims jwt.MapClaims) error`
    - `validateAudience(claims jwt.MapClaims) error`
    - `validateRequiredClaims(claims jwt.MapClaims) error`
  - Implement time-based validation with `ClockSkew`:
    - `exp` (expiration): `now > exp + skew` → error
    - `nbf` (not before): `now < nbf - skew` → error
    - `iat` (issued at): `now < iat - skew` → error
  - Handle optional issuer and audience validation
  - Validate required claims existence and non-empty values
  - _Requirements: FR3, FR4, FR5_

  - [x] 10.1 Create claims validation unit tests
    - Test expired token rejection
    - Test not-yet-valid token rejection (nbf)
    - Test invalid issued-at time
    - Test clock skew tolerance
    - Test issuer validation (matching and non-matching)
    - Test audience validation (single and multiple)
    - Test required claims validation
    - Test missing required claims rejection
    - _Requirements: FR3, FR4, FR5_

- [x] 11. Implement algorithm whitelist validation
  - Add `isAlgorithmAllowed(alg string) bool` method to `TokenValidator`
  - If `config.AllowedAlgorithms` is empty, allow all algorithms
  - If configured, only allow algorithms in the whitelist
  - Return error for disallowed algorithms
  - _Requirements: FR2_

  - [x] 11.1 Create algorithm validation unit tests
    - Test allowed algorithm acceptance
    - Test disallowed algorithm rejection
    - Test no whitelist allows all
    - Test algorithm confusion attack prevention
    - _Requirements: FR2, NFR2_

- [x] 12. Generate test JWTs for unit testing
  - Create `testutil/` package with JWT generation helpers
  - Implement `GenerateHMACTestKey() []byte`
  - Implement `GenerateRSATestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey)`
  - Implement `GenerateECDSATestKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey)`
  - Implement `GenerateValidJWT(key interface{}, claims map[string]interface{}) string`
  - Implement `GenerateExpiredJWT(key interface{}) string`
  - Implement `GenerateInvalidSignatureJWT() string`
  - Use these helpers across unit tests
  - _Requirements: NFR4_

- [x] 13. Create static key mode integration test
  - Create `jwt_test.go` with end-to-end tests
  - Test complete validation flow for HMAC, RSA, ECDSA
  - Test error cases (expired, invalid signature, missing claims)
  - Test claims extraction and context storage
  - _Requirements: FR2, FR3, FR4, FR5, FR6_

### Milestone 3: JWKS Support

- [x] 14. Implement JWKS cache
  - Create `jwks.go` with `JWKSCache` struct:
    - `keys sync.Map` (kid → crypto.PublicKey)
    - `lastFetch time.Time`
    - `ttl time.Duration`
    - `mu sync.RWMutex` (for lastFetch access)
  - Implement `NewJWKSCache(ttl time.Duration) *JWKSCache`
  - Implement `Get(kid string) (interface{}, bool)`:
    - Check if cache is stale (`time.Since(lastFetch) > ttl`)
    - Return `nil, false` if stale
    - Load key from `sync.Map` and return
  - Implement `Set(kid string, key interface{})`:
    - Store key in `sync.Map`
  - Implement `UpdateLastFetch()`:
    - Update `lastFetch` timestamp (thread-safe)
  - _Requirements: FR8_

  - [x] 14.1 Create JWKS cache unit tests
    - Test concurrent Get/Set operations (use `-race` flag)
    - Test cache expiration behavior
    - Test cache hit/miss scenarios
    - Test UpdateLastFetch updates timestamp
    - _Requirements: FR8, NFR3_

- [x] 15. Implement JWKS fetcher
  - Create `fetchJWKS(ctx context.Context, endpoint string, httpClient *http.Client) (map[string]crypto.PublicKey, error)` function
  - Create HTTP GET request to JWKS endpoint
  - Parse JSON response as JWKS structure
  - Use `github.com/lestrrat-go/jwx/v2/jwk` to parse JWK keys
  - Extract `kid` and convert JWK to `crypto.PublicKey` for each key
  - Return map of `kid` → `PublicKey`
  - Handle HTTP errors (timeout, non-200 status, invalid JSON)
  - _Requirements: FR7_

  - [x] 15.1 Create JWKS fetcher unit tests
    - Test successful JWKS fetch and parsing
    - Test HTTP error handling (404, 500, timeout)
    - Test invalid JSON response
    - Test malformed JWK data
    - Mock HTTP server for controlled testing
    - _Requirements: FR7_

- [x] 16. Implement JWKS key provider
  - Create `JWKSKeyProvider` struct:
    - `endpoint string`
    - `cache *JWKSCache`
    - `httpClient *http.Client`
    - `refreshMu sync.Mutex` (prevent concurrent refreshes)
  - Implement `NewJWKSKeyProvider(endpoint string, cache *JWKSCache, timeout time.Duration) *JWKSKeyProvider`
  - Implement `GetKey(ctx context.Context, kid string) (interface{}, error)`:
    - Try `cache.Get(kid)` first
    - If cache hit, return key
    - If cache miss, call `RefreshCache(ctx)`
    - Retry `cache.Get(kid)` after refresh
    - Return error if still not found
  - Implement `RefreshCache(ctx context.Context) error`:
    - Use `refreshMu` to prevent concurrent refreshes
    - Call `fetchJWKS()` to get keys
    - Update cache with all fetched keys
    - Call `cache.UpdateLastFetch()`
    - Log refresh success/failure
  - _Requirements: FR7, FR8_

  - [x] 16.1 Create JWKS key provider unit tests
    - Test GetKey with cache hit
    - Test GetKey with cache miss and successful refresh
    - Test GetKey with refresh failure
    - Test concurrent GetKey calls (deduplication)
    - Test RefreshCache updates cache correctly
    - Mock HTTP client and JWKS responses
    - _Requirements: FR7, FR8_

- [x] 17. Implement refresh-on-signature-failure strategy
  - Update `TokenValidator.Validate()` to handle signature failures:
    - If parsing fails with signature error AND using JWKSKeyProvider
    - Force cache refresh by calling provider's `RefreshCache()` directly
    - Retry JWT parsing once after refresh
    - If still fails, return signature error
  - Add method to force refresh in `JWKSKeyProvider` if needed
  - _Requirements: FR8_

  - [x] 17.1 Create refresh-on-failure unit tests
    - Test signature failure triggers JWKS refresh
    - Test successful validation after refresh
    - Test failure persists after refresh (invalid token)
    - Mock JWKS provider with updated keys
    - _Requirements: FR8_

- [x] 18. Implement background JWKS refresh
  - Add background refresh logic to `Middleware`:
    - Start goroutine in `Start()` if `JWKSRefreshInterval > 0`
    - Use `time.Ticker` for periodic refresh
    - Call `provider.RefreshCache()` on each tick
    - Log success/failure of background refresh
    - Stop goroutine on `Stop()` using context cancellation
  - Add context fields to `Middleware`:
    - `refreshCtx context.Context`
    - `refreshCancel context.CancelFunc`
  - _Requirements: FR8_

  - [x] 18.1 Create background refresh unit tests
    - Test background refresh goroutine starts
    - Test periodic refresh executes
    - Test refresh stops on middleware stop
    - Test no goroutine leaks
    - _Requirements: FR8, NFR7_

- [x] 19. Create JWKS mode integration test
  - Create mock JWKS HTTP server
  - Generate test JWKS with multiple keys (different `kid`)
  - Test full JWKS flow: startup fetch, cache, validation, refresh
  - Test key rotation scenario (JWKS returns new keys)
  - Test signature failure and refresh behavior
  - Test background refresh (if configured)
  - _Requirements: FR7, FR8_

### Milestone 4: Mono Middleware Integration

- [x] 20. Implement Middleware struct and constructor
  - Create `jwt.go` with `Middleware` struct:
    - `name string`
    - `config *Config`
    - `validator *TokenValidator`
    - `jwksCache *JWKSCache` (nil if using static keys)
    - `logger *slog.Logger`
    - `refreshCtx context.Context`
    - `refreshCancel context.CancelFunc`
  - Implement `New(opts ...Option) (*Middleware, error)`:
    - Create default config with sensible defaults
    - Apply all functional options
    - Validate configuration
    - Initialize appropriate key provider (static vs JWKS)
    - Initialize validator
    - Return middleware instance
  - _Requirements: FR11, NFR1_

  - [x] 20.1 Create middleware constructor unit tests
    - Test successful creation with all key source types
    - Test error when no key source provided
    - Test error when multiple key sources provided
    - Test default values are applied
    - _Requirements: FR11_

- [x] 21. Implement mono.MiddlewareModule interface methods
  - Implement `Name() string`:
    - Return `"jwt"`
  - Implement `Logger() *slog.Logger`:
    - Return `m.logger` (set by Mono framework after registration)
  - Implement `Start(ctx context.Context) error`:
    - If using JWKS, perform initial fetch (fail fast if fails)
    - Start background refresh goroutine if `JWKSRefreshInterval > 0`
    - Log startup success with mode (static/jwks)
  - Implement `Stop(ctx context.Context) error`:
    - Cancel background refresh goroutine (if running)
    - Wait for goroutine to exit
    - Log shutdown
  - _Requirements: NFR1_

  - [x] 21.1 Create lifecycle methods unit tests
    - Test Start() performs initial JWKS fetch
    - Test Start() starts background refresh
    - Test Stop() cancels background refresh
    - Test Stop() waits for goroutine exit
    - Mock JWKS provider for controlled testing
    - _Requirements: NFR1, NFR7_

- [x] 22. Implement OnServiceRegistration hook
  - Implement `OnServiceRegistration(ctx context.Context, reg types.ServiceRegistration) types.ServiceRegistration`:
    - Check if service should be skipped using `shouldSkip(reg.ModuleName, reg.ServiceName)`
    - Wrap `RequestReplyHandler` if present
    - Wrap `QueueGroupHandler` if present
    - Wrap `StreamConsumerHandler` if present
    - Wrap `EventConsumerHandler` if present
    - Wrap `EventStreamConsumerHandler` if present
    - Return modified registration
  - Implement `shouldSkip(moduleName, serviceName string) bool`:
    - Check if `moduleName.serviceName`, `moduleName`, or `serviceName` matches any `SkipPaths`
    - Return true if match found
  - Reference `v1/otel/` for handler wrapping pattern
  - _Requirements: FR9, FR10, NFR1_

  - [x] 22.1 Create OnServiceRegistration unit tests
    - Test RequestReplyHandler wrapping
    - Test all 5 handler types are wrapped
    - Test skip paths logic
    - Test registration passthrough when skipped
    - _Requirements: FR9, FR10_

- [x] 23. Implement handler wrapper for RequestReplyHandler
  - Create `wrapRequestReplyHandler(original types.RequestReplyHandler, moduleName, serviceName string) types.RequestReplyHandler`:
    - Extract token from `msg.Header` using `extractToken()`
    - If extraction fails and `config.Optional` is true, call original handler without validation
    - If extraction fails and `config.Optional` is false, return error
    - Validate token using `validator.Validate(ctx, token)`
    - If validation fails, log warning and return error
    - If validation succeeds, add claims to context using `WithClaims()`
    - Call original handler with enhanced context
    - Log validation success at DEBUG level
  - _Requirements: FR1, FR2, FR3, FR4, FR5, FR6, FR9_

  - [x] 23.1 Create RequestReplyHandler wrapper unit tests
    - Test valid token passes through to handler
    - Test invalid token prevents handler execution
    - Test claims are available in context
    - Test optional mode allows missing token
    - Test logging of validation events
    - _Requirements: FR1, FR6, FR9_

- [ ] 24. Implement handler wrappers for other handler types
  - Create `wrapQueueGroupHandler()` with same pattern as RequestReply
  - Create `wrapStreamConsumerHandler()` with same pattern
  - Create `wrapEventConsumerHandler()` with same pattern
  - Create `wrapEventStreamConsumerHandler()` with same pattern
  - Each wrapper follows the same validation flow:
    - Extract token → Validate → Add claims to context → Call original
  - _Requirements: FR9_

  - [ ] 24.1 Create wrapper unit tests for all handler types
    - Test QueueGroupHandler wrapper
    - Test StreamConsumerHandler wrapper
    - Test EventConsumerHandler wrapper
    - Test EventStreamConsumerHandler wrapper
    - Verify same validation behavior across all types
    - _Requirements: FR9_

- [ ] 25. Implement token extraction from message headers
  - Update `extractToken()` to work with `map[string][]string` (Mono message headers)
  - Perform case-insensitive header key lookup (Mono headers are case-insensitive)
  - Support `config.HeaderKey` and `config.TokenPrefix` customization
  - Handle header values as string slices (take first value)
  - _Requirements: FR1, IR2_

  - [ ] 25.1 Create message header extraction unit tests
    - Test extraction from Mono message structure
    - Test case-insensitive header lookup
    - Test custom header key and prefix
    - Test header with multiple values (use first)
    - _Requirements: FR1, IR2_

- [ ] 26. Create Mono integration test with all handler types
  - Create test Mono application
  - Create test modules with all 5 handler types
  - Register JWT middleware
  - Test each handler type with:
    - Valid token (200/success)
    - Invalid token (error)
    - Expired token (error)
    - Missing token (error, unless optional)
    - Claims available in context
  - Test skip paths functionality
  - _Requirements: FR9, FR10, NFR1_

### TODO
### Milestone 5: Testing & Documentation

- [ ] 27. Create comprehensive security test suite
  - Create `security_test.go` with security-focused tests
  - Test token tampering detection (modify payload after signing)
  - Test algorithm confusion attack (switch from RS256 to HS256)
  - Test key confusion with JWKS (wrong `kid`)
  - Test timing attack resistance (verify constant-time operations)
  - Test replay attack scenario (same token used multiple times - should succeed unless expired)
  - Verify no sensitive info in error messages (no token/key leakage)
  - Test all error paths return appropriate errors
  - _Requirements: NFR2, NFR4_

- [ ] 28. Create performance benchmarks
  - Create `benchmark_test.go` with performance benchmarks
  - Benchmark `Validate()` with static secret (target: <10ms)
  - Benchmark `Validate()` with static public key (target: <10ms)
  - Benchmark `Validate()` with JWKS cache hit (target: <20ms)
  - Benchmark `Validate()` with JWKS cache miss (measures refresh overhead)
  - Benchmark concurrent validations (10, 100, 1000 goroutines)
  - Benchmark JWKS cache operations (Get, Set)
  - Verify memory allocations are minimal
  - Run with `-race` flag to detect race conditions
  - _Requirements: NFR3, NFR4_

- [ ] 29. Create example applications
  - Create `examples/` directory with sample Mono applications:
    - `examples/static-secret/` - HMAC with static secret
    - `examples/static-publickey/` - RSA/ECDSA with static public key
    - `examples/jwks-mode/` - JWKS endpoint mode
    - `examples/skip-paths/` - Skip paths configuration
    - `examples/claims-usage/` - Accessing claims in handlers (all 5 types)
  - Each example should have:
    - `main.go` with working Mono application
    - `README.md` with explanation and instructions
    - Sample JWT tokens for testing
  - _Requirements: NFR5_

- [ ] 30. Create comprehensive README.md
  - Write main `README.md` in package root:
    - Overview of features and use cases
    - Installation: `go get github.com/go-monolith/contrib/v1/jwt@latest`
    - Quick start guide (static secret mode)
    - Configuration reference (all options)
    - JWKS mode setup guide
    - Claims extraction and usage examples
    - Skip paths configuration
    - Handler wrapping for all types
    - Security best practices
    - Performance characteristics
    - Troubleshooting guide (common errors and solutions)
    - Contributing guidelines
    - License information
  - Include code examples inline
  - Link to `examples/` directory
  - _Requirements: NFR5_

- [ ] 31. Add godoc comments to all exports
  - Review all exported types, functions, and constants
  - Add comprehensive godoc comments:
    - Package-level comment in `jwt.go`
    - Type documentation for `Middleware`, `Config`, etc.
    - Function documentation with parameters and return values
    - Usage examples for complex functions
    - Error conditions and edge cases
  - Run `go doc` locally to verify rendering
  - _Requirements: NFR5_

- [ ] 32. Create CI/CD pipeline configuration
  - Create `.github/workflows/ci.yml` for GitHub Actions:
    - Run tests on Go 1.21, 1.22, 1.23
    - Run tests on Linux, macOS, Windows
    - Run race detector: `go test -race ./...`
    - Run coverage: `go test -coverprofile=coverage.out ./...`
    - Upload coverage to codecov.io
    - Run linters: `golangci-lint run`
    - Verify go.mod is tidy
    - Check for known CVEs: `go list -m all | nancy sleuth`
  - Create `.golangci.yml` with linter configuration
  - _Requirements: NFR2, NFR4_

- [ ] 33. Create migration guide from existing REQUIREMENTS.md
  - Document differences between initial requirements and final implementation
  - Create checklist mapping original acceptance criteria to implementation
  - Add notes for any deviations or enhancements
  - Keep original `REQUIREMENTS.md` as reference
  - _Requirements: NFR5_

---

## Optional Enhancements (Post-MVP)

- [ ] Support for JWT ID (`jti`) claim validation (prevent replay attacks)
- [ ] Distributed JWKS cache (Redis) for multi-instance deployments
- [ ] Prometheus metrics for validation latency, cache hit rate, errors
- [ ] Rate limiting for JWKS endpoint requests
- [ ] Support for custom claim validators (callback functions)
- [ ] Support for multiple JWKS endpoints (failover)
- [ ] Support for Ed25519/EdDSA algorithm
- [ ] CLI tool for JWT generation and validation (testing utility)
- [ ] WebHook for JWKS key rotation notifications

---

## Testing Checklist

- [ ] Unit tests for all components (>80% coverage)
- [ ] Integration tests with Mono application
- [ ] Security tests (tampering, algorithm confusion, etc.)
- [ ] Performance benchmarks (<10ms static, <20ms JWKS)
- [ ] Race detector tests (no data races)
- [ ] JWKS mock server tests
- [ ] All 5 handler types tested
- [ ] Error path tests (all error types)
- [ ] Context helpers tests
- [ ] Configuration validation tests

---

*Last Updated: January 2026*
*Version: 1.0 (MVP Scope)*
