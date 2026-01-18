## Summary

Develop a JWT authentication middleware for the Mono framework located in `v1/jwt/` that automatically validates the `Authorization` header in `mono.Message.Header`, extracts JWT claims, and sets them in the request `context.Context`.

## Background

Following the established middleware patterns in this project (see `v1/otel/` for reference), we need a JWT middleware that integrates seamlessly with the Mono framework's handler wrapping approach.

## Requirements

### Core Functionality

1. **Header Extraction**

   - Read the `Authorization` header from `msg.Header`
   - Support `Bearer <token>` format
   - Handle case-insensitive header keys

2. **JWT Validation**

   - Validate token signature
   - Validate standard claims (exp, nbf, iat, iss, aud)
   - Support both symmetric (HS256, HS384, HS512) and asymmetric (RS256, RS384, RS512, ES256, ES384, ES512) algorithms

3. **Context Enhancement**

   - Extract validated claims from JWT payload
   - Set claims in `context.Context` using a well-defined context key
   - Provide helper functions to retrieve claims from context

4. **Error Handling**
   - Return appropriate errors for:
     - Missing Authorization header
     - Invalid token format
     - Expired token
     - Invalid signature
     - Unsupported algorithm
   - Do not proceed with handler execution on validation failure

### Key Source Configuration

Support two modes of key configuration:

#### Option A: Pre-configured Keys

- Static `PublicKey` for asymmetric algorithms (RSA, ECDSA)
- Static `Secret` for symmetric algorithms (HMAC)
- Support PEM-encoded keys

#### Option B: JWKS Endpoint

- Dynamically retrieve public keys from a JWKS (JSON Web Key Set) endpoint
- **Caching Requirements:**
  - Cache keys to avoid frequent network requests
  - Configurable cache TTL (default: 1 hour)
  - Support cache invalidation on key rotation (kid mismatch)
  - Background refresh before TTL expiration
- Handle key rotation gracefully by matching `kid` (Key ID) claim

## Technical Design

### Directory Structure

```
v1/jwt/
├── jwt.go              # Main middleware implementation (mono.MiddlewareModule)
├── config.go           # Configuration struct and DefaultConfig()
├── options.go          # Functional options for configuration
├── validator.go        # JWT validation logic
├── jwks.go             # JWKS fetcher and cache
├── context.go          # Context key definitions and helper functions
├── errors.go           # Custom error types
├── jwt_test.go         # Unit tests for middleware
├── validator_test.go   # Unit tests for validation
├── jwks_test.go        # Unit tests for JWKS (with mocked HTTP)
├── context_test.go     # Unit tests for context helpers
├── go.mod              # Module definition
└── README.md           # Documentation with usage examples
```

### Middleware Interface Implementation

```go
// Implements mono.MiddlewareModule
type Middleware struct {
    name       string
    config     *Config
    validator  *JWTValidator
    jwksCache  *JWKSCache  // nil if using static keys
    logger     *slog.Logger
}

func (m *Middleware) Name() string
func (m *Middleware) Start(ctx context.Context) error
func (m *Middleware) Stop(ctx context.Context) error
func (m *Middleware) OnServiceRegistration(ctx context.Context, reg types.ServiceRegistration) types.ServiceRegistration
func (m *Middleware) Logger() *slog.Logger
// ... other hook methods (can return unchanged for non-applicable hooks)
```

### Handler Wrapping Pattern

```go
func (m *Middleware) wrapRequestReplyHandler(
    original types.RequestReplyHandler,
    moduleName, serviceName string,
) types.RequestReplyHandler {
    return func(ctx context.Context, msg *types.Msg) ([]byte, error) {
        // 1. Extract Authorization header
        token, err := m.extractBearerToken(msg.Header)
        if err != nil {
            return nil, err
        }

        // 2. Validate JWT and extract claims
        claims, err := m.validator.Validate(ctx, token)
        if err != nil {
            return nil, err
        }

        // 3. Add claims to context
        ctx = WithClaims(ctx, claims)

        // 4. Call original handler with enhanced context
        return original(ctx, msg)
    }
}
```

### Configuration Options

```go
type Config struct {
    // Key source options (mutually exclusive)
    Secret     []byte            // For HMAC algorithms
    PublicKey  crypto.PublicKey  // For RSA/ECDSA algorithms
    JWKSEndpoint string          // URL for JWKS endpoint

    // JWKS cache settings
    JWKSCacheTTL          time.Duration  // Default: 1 hour
    JWKSRefreshInterval   time.Duration  // Default: 50 minutes (before TTL)
    JWKSRequestTimeout    time.Duration  // Default: 10 seconds

    // Validation options
    RequiredClaims    []string          // Claims that must be present
    ExpectedIssuer    string            // Expected 'iss' claim
    ExpectedAudience  []string          // Expected 'aud' claim(s)
    AllowedAlgorithms []string          // Whitelist of allowed algorithms
    ClockSkew         time.Duration     // Default: 1 minute

    // Header settings
    HeaderKey         string            // Default: "authorization"
    TokenPrefix       string            // Default: "Bearer "

    // Behavior settings
    SkipPaths         []string          // Service paths to skip validation
    Optional          bool              // If true, allow requests without token
}
```

### Context Helper Functions

```go
// Context key type for claims
type claimsContextKey struct{}

// WithClaims adds JWT claims to context
func WithClaims(ctx context.Context, claims jwt.MapClaims) context.Context

// ClaimsFromContext retrieves JWT claims from context
func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool)

// MustClaimsFromContext retrieves claims or panics
func MustClaimsFromContext(ctx context.Context) jwt.MapClaims

// SubjectFromContext is a convenience function to get the 'sub' claim
func SubjectFromContext(ctx context.Context) (string, bool)
```

### JWKS Cache Implementation

```go
type JWKSCache struct {
    endpoint      string
    httpClient    *http.Client
    keys          sync.Map       // kid -> *rsa.PublicKey or *ecdsa.PublicKey
    lastFetch     time.Time
    ttl           time.Duration
    refreshTicker *time.Ticker
    mu            sync.RWMutex
}

func (c *JWKSCache) GetKey(ctx context.Context, kid string) (crypto.PublicKey, error)
func (c *JWKSCache) Refresh(ctx context.Context) error
func (c *JWKSCache) Start(ctx context.Context) error  // Start background refresh
func (c *JWKSCache) Stop(ctx context.Context) error   // Stop background refresh
```

## Usage Examples

### With Static Secret (HMAC)

```go
import "github.com/go-monolith/contrib/v1/jwt"

jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("your-256-bit-secret")),
    jwt.WithExpectedIssuer("https://your-issuer.com"),
    jwt.WithExpectedAudience("your-audience"),
)
if err != nil {
    log.Fatal(err)
}

app.Register(jwtMw)
```

### With Static Public Key (RSA/ECDSA)

```go
import "github.com/go-monolith/contrib/v1/jwt"

publicKey, _ := jwt.ParseRSAPublicKeyFromPEM(pemData)

jwtMw, err := jwt.New(
    jwt.WithPublicKey(publicKey),
    jwt.WithAllowedAlgorithms([]string{"RS256", "RS384", "RS512"}),
)
```

### With JWKS Endpoint

```go
import "github.com/go-monolith/contrib/v1/jwt"

jwtMw, err := jwt.New(
    jwt.WithJWKSEndpoint("https://your-auth-server.com/.well-known/jwks.json"),
    jwt.WithJWKSCacheTTL(2 * time.Hour),
    jwt.WithJWKSRefreshInterval(90 * time.Minute),
)
```

### Accessing Claims in Handler

```go
func (m *MyModule) HandleRequest(ctx context.Context, msg *types.Msg) ([]byte, error) {
    // Get all claims
    claims, ok := jwt.ClaimsFromContext(ctx)
    if !ok {
        return nil, errors.New("no claims in context")
    }

    // Get specific claim
    userID := claims["sub"].(string)

    // Or use helper
    subject, ok := jwt.SubjectFromContext(ctx)

    // ... handle request
}
```

## Acceptance Criteria

- [ ] Middleware implements `mono.MiddlewareModule` interface
- [ ] Validates JWT signature using configured key source
- [ ] Validates standard JWT claims (exp, nbf, iat)
- [ ] Supports optional issuer and audience validation
- [ ] Extracts claims and adds to `context.Context`
- [ ] Supports HMAC algorithms (HS256, HS384, HS512)
- [ ] Supports RSA algorithms (RS256, RS384, RS512)
- [ ] Supports ECDSA algorithms (ES256, ES384, ES512)
- [ ] JWKS endpoint support with caching
- [ ] Background JWKS refresh before cache expiration
- [ ] Graceful handling of key rotation (kid matching)
- [ ] Wraps all handler types (RequestReply, QueueGroup, StreamConsumer, EventConsumer, EventStreamConsumer)
- [ ] Proper error messages for all failure cases
- [ ] Comprehensive unit tests with >80% coverage
- [ ] Integration tests with mocked JWKS server
- [ ] Documentation with usage examples

## Dependencies

- `github.com/golang-jwt/jwt/v5` - JWT parsing and validation
- Standard library only for JWKS fetching (no external HTTP client dependencies)

## Related

- Reference implementation: `v1/otel/` (OpenTelemetry middleware)
- Mono framework: `github.com/go-monolith/mono`

## Labels

`enhancement`, `middleware`, `security`, `authentication`
