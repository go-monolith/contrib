# JWT Authentication Middleware for Mono Framework

A comprehensive JWT authentication middleware for the [Mono framework](https://github.com/go-monolith/mono) that validates JWT tokens in NATS message headers and enriches the request context with validated claims.

## Features

- **JWT Signature Verification**: Support for HMAC (HS256/HS384/HS512), RSA (RS256/RS384/RS512), and ECDSA (ES256/ES384/ES512) algorithms
- **Standard Claims Validation**: Automatic validation of `exp`, `nbf`, `iat` claims with configurable clock skew
- **Issuer & Audience Validation**: Optional validation of `iss` and `aud` claims
- **JWKS Endpoint Support**: Dynamic key fetching from JWKS endpoints with intelligent caching and automatic key rotation handling
- **Context Enhancement**: Validated claims automatically injected into `context.Context` for downstream handlers
- **All Handler Types**: Wraps RequestReply, QueueGroup, and EventConsumer handlers automatically. Batch handlers (StreamConsumer, EventStreamConsumer) require manual validation using the exposed `Validator()` method for security
- **Skip Paths**: Flexible configuration to exclude specific services from JWT validation
- **Optional Mode**: Allow requests without tokens for public endpoints
- **High Performance**: 89.8% test coverage, thread-safe, optimized for high throughput
- **Production Ready**: Comprehensive error handling, structured logging, and resource management

## Installation

```bash
go get github.com/go-monolith/contrib/v1/jwt
```

## Quick Start

### Static Secret (HMAC)

Use this mode when you have a shared secret key for HS256/HS384/HS512 tokens:

```go
package main

import (
    "log"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/contrib/v1/jwt"
)

func main() {
    // Create JWT middleware with HMAC secret
    jwtMw, err := jwt.New(
        jwt.WithSecret([]byte("your-256-bit-secret")),
        jwt.WithExpectedIssuer("https://your-issuer.com"),
        jwt.WithExpectedAudience("your-audience"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create Mono application and register middleware
    app := mono.New()
    app.Register(jwtMw)

    // Register your modules...
    // app.Register(yourModule)

    // Start the application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Static Public Key (RSA/ECDSA)

Use this mode when you have a public key for RS*/ES* tokens:

```go
package main

import (
    "log"
    "os"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/contrib/v1/jwt"
)

func main() {
    // Load PEM-encoded public key
    pemData, err := os.ReadFile("public_key.pem")
    if err != nil {
        log.Fatal(err)
    }

    // Parse RSA public key
    publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pemData)
    if err != nil {
        log.Fatal(err)
    }

    // Create JWT middleware with public key
    jwtMw, err := jwt.New(
        jwt.WithPublicKey(publicKey),
        jwt.WithAllowedAlgorithms("RS256", "RS384", "RS512"),
        jwt.WithExpectedIssuer("https://your-issuer.com"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create Mono application and register middleware
    app := mono.New()
    app.Register(jwtMw)

    // Start the application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### JWKS Endpoint (Dynamic Key Fetching)

Use this mode for Auth0, Keycloak, or other OAuth2/OIDC providers with key rotation:

```go
package main

import (
    "log"
    "time"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/contrib/v1/jwt"
)

func main() {
    // Create JWT middleware with JWKS endpoint
    jwtMw, err := jwt.New(
        jwt.WithJWKSEndpoint("https://your-auth-server.com/.well-known/jwks.json"),
        jwt.WithJWKSCacheTTL(2 * time.Hour),
        jwt.WithJWKSRefreshInterval(90 * time.Minute),
        jwt.WithExpectedIssuer("https://your-auth-server.com/"),
        jwt.WithExpectedAudience("your-api-audience"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create Mono application and register middleware
    app := mono.New()
    app.Register(jwtMw)

    // Start the application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

### Key Source Options (Mutually Exclusive)

| Option | Description | Example |
|--------|-------------|---------|
| `WithSecret([]byte)` | HMAC secret for HS256/HS384/HS512 | `jwt.WithSecret([]byte("secret"))` |
| `WithPublicKey(crypto.PublicKey)` | RSA/ECDSA public key for RS*/ES* | `jwt.WithPublicKey(rsaKey)` |
| `WithJWKSEndpoint(string)` | JWKS URL for dynamic key fetching | `jwt.WithJWKSEndpoint("https://...")` |

### JWKS Settings (Only for JWKS mode)

| Option | Default | Description |
|--------|---------|-------------|
| `WithJWKSCacheTTL(duration)` | 1 hour | How long to cache JWKS keys |
| `WithJWKSRefreshInterval(duration)` | 50 minutes | Background refresh interval (0 to disable) |

### Validation Settings

| Option | Default | Description |
|--------|---------|-------------|
| `WithExpectedIssuer(string)` | "" (skip) | Expected "iss" claim value |
| `WithExpectedAudience(...string)` | [] (skip) | Expected "aud" claim values (at least one must match) |
| `WithRequiredClaims(...string)` | [] | Claims that must be present |
| `WithAllowedAlgorithms(...string)` | all matching key type | Whitelist of allowed signing algorithms |
| `WithClockSkew(duration)` | 1 minute | Clock drift tolerance for time-based claims |

### Behavior Settings

| Option | Default | Description |
|--------|---------|-------------|
| `WithSkipPaths(...string)` | [] | Service paths to skip validation (see Skip Paths section) |
| `WithOptional(bool)` | false | Allow requests without tokens to proceed |

## Usage Examples

### Example 1: RequestReply Handler

```go
package mymodule

import (
    "context"
    "encoding/json"

    "github.com/go-monolith/mono/pkg/types"
    "github.com/go-monolith/contrib/v1/jwt"
)

type MyModule struct{}

func (m *MyModule) Name() string {
    return "mymodule"
}

func (m *MyModule) RegisterServices(reg types.ServiceRegistry) error {
    return reg.RequestReply("GetUser", m.HandleGetUser)
}

func (m *MyModule) HandleGetUser(ctx context.Context, msg *types.Msg) ([]byte, error) {
    // Extract claims from context
    claims, ok := jwt.ClaimsFromContext(ctx)
    if !ok {
        return nil, errors.New("no claims in context")
    }

    // Get user ID from subject claim
    userID, ok := jwt.SubjectFromContext(ctx)
    if !ok {
        return nil, errors.New("no subject in token")
    }

    // Access custom claims
    email := claims["email"].(string)
    role := claims["role"].(string)

    // Build response
    response := map[string]interface{}{
        "user_id": userID,
        "email":   email,
        "role":    role,
    }

    return json.Marshal(response)
}
```

### Example 2: QueueGroup Handler

```go
func (m *MyModule) RegisterServices(reg types.ServiceRegistry) error {
    return reg.QueueGroup("ProcessTask", []types.QGHP{
        {QueueGroup: "workers", Handler: m.HandleProcessTask},
    })
}

func (m *MyModule) HandleProcessTask(ctx context.Context, msg *types.Msg) error {
    // Get subject (user ID) from context
    userID, ok := jwt.SubjectFromContext(ctx)
    if !ok {
        return errors.New("authentication required")
    }

    // Process task with authenticated user context
    log.Printf("Processing task for user: %s", userID)

    return nil
}
```

### Example 3: StreamConsumer Handler (Manual Validation Required)

**Note:** Batch handlers (StreamConsumer, EventStreamConsumer) are NOT automatically wrapped by the middleware because validating only the first message would be insecure - each message may have a different authentication context. You must validate each message individually using the exposed `Extract()` and `Validator()` method.

```go
type MyModule struct {
    jwtValidator *jwt.TokenValidator
}

// Pass the JWT middleware.Validator() as `validator` to your module constructor
func NewMyModule(validator *jwt.TokenValidator) *MyModule {
    return &MyModule{
        jwtValidator: validator,
    }
}

func (m *MyModule) RegisterServices(reg types.ServiceRegistry) error {
    return reg.StreamConsumer("ProcessBatch", "my-stream", m.HandleProcessBatch)
}

func (m *MyModule) HandleProcessBatch(ctx context.Context, msgs []*types.Msg) error {
    // Validate each message individually
    for _, msg := range msgs {
        // Extract token from message
        token, err := m.jwtValidator.Extract(msg)
        if err != nil {
            log.Printf("Skipping message without valid token: %v", err)
            continue // or return err to fail the entire batch
        }

        // Validate token
        claims, err := m.jwtValidator.Validate(ctx, token)
        if err != nil {
            log.Printf("Skipping message with invalid token: %v", err)
            continue // or return err to fail the entire batch
        }

        // Process message with authenticated claims
        userID := claims["sub"].(string)
        log.Printf("Processing message for user: %s", userID)
        // ... process message
    }

    return nil
}
```

### Example 4: EventConsumer Handler

```go
func (m *MyModule) RegisterEventConsumers(bus types.EventBus) error {
    return bus.Subscribe(types.EventDef{
        Name: "UserCreated",
    }, m.HandleUserCreated)
}

func (m *MyModule) HandleUserCreated(ctx context.Context, msg *types.Msg) error {
    // Extract claims from context
    claims := jwt.MustClaimsFromContext(ctx) // Panics if not found

    adminID := claims["sub"].(string)

    // Process event with admin context
    log.Printf("User created by admin: %s", adminID)

    return nil
}
```

### Example 5: EventStreamConsumer Handler (Manual Validation Required)

**Note:** Like StreamConsumer, EventStreamConsumer handlers require manual validation of each message to ensure security. Use the exposed `Extract()` and `Validator()` method.

```go
type MyModule struct {
    jwtValidator *jwt.TokenValidator
}

// Pass the JWT middleware.Validator() as `validator` to your module constructor
func NewMyModule(validator *jwt.TokenValidator) *MyModule {
    return &MyModule{
        jwtValidator: validator,
    }
}

func (m *MyModule) RegisterEventConsumers(bus types.EventBus) error {
    return bus.StreamSubscribe(types.EventDef{
        Name: "OrderPlaced",
    }, "order-processor", m.HandleOrderBatch)
}

func (m *MyModule) HandleOrderBatch(ctx context.Context, msgs []*types.Msg) error {
    // Validate each message individually
    for _, msg := range msgs {
        // Extract token from message
        token, err := m.jwtValidator.Extract(msg)
        if err != nil {
            log.Printf("Skipping event without valid token: %v", err)
            continue // or return err to fail the entire batch
        }

        // Validate token
        claims, err := m.jwtValidator.Validate(ctx, token)
        if err != nil {
            log.Printf("Skipping event with invalid token: %v", err)
            continue // or return err to fail the entire batch
        }

        // Process event with authenticated claims
        userID := claims["sub"].(string)
        log.Printf("Processing order for user: %s", userID)
        // ... process order
    }

    return nil
}
```

### Example 6: Skip Paths for Public Endpoints

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithSkipPaths(
        "health.Check",     // Skip "Check" service in "health" module
        "auth.Login",       // Skip "Login" service in "auth" module
        "auth.Register",    // Skip "Register" service in "auth" module
        "metrics",          // Skip all services in "metrics" module
    ),
)
```

**Skip Path Matching Rules:**
- `"module.service"` - Exact match of module and service name
- `"module"` - Skip all services in the specified module
- `"service"` - Skip this service name in all modules

### Example 7: Optional Mode for Mixed Public/Private Endpoints

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithOptional(true), // Allow requests without tokens
)

// In your handler:
func (m *MyModule) HandleGetContent(ctx context.Context, msg *types.Msg) ([]byte, error) {
    claims, ok := jwt.ClaimsFromContext(ctx)
    if ok {
        // Authenticated user - return premium content
        userID := claims["sub"].(string)
        return getPremiumContent(userID)
    } else {
        // Anonymous user - return public content
        return getPublicContent()
    }
}
```

### Example 8: Required Claims Validation

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithRequiredClaims("sub", "email", "role"),
    jwt.WithExpectedIssuer("https://my-auth.com"),
    jwt.WithExpectedAudience("my-api", "my-admin-api"),
)

// All tokens must have sub, email, and role claims
// Tokens missing any required claim will be rejected
```

## Context Helper Functions

The middleware provides convenient helper functions to extract claims from context:

```go
// Get all claims as jwt.MapClaims
claims, ok := jwt.ClaimsFromContext(ctx)
if !ok {
    // No claims in context (optional mode or public endpoint)
}

// Get all claims (panics if not found - use only when JWT is required)
claims := jwt.MustClaimsFromContext(ctx)

// Get subject (user ID) claim
userID, ok := jwt.SubjectFromContext(ctx)
if !ok {
    // No subject claim
}

// Get issuer claim
issuer, ok := jwt.IssuerFromContext(ctx)
if !ok {
    // No issuer claim
}

// Access custom claims
email := claims["email"].(string)
role := claims["role"].(string)
permissions := claims["permissions"].([]interface{})
```

## PEM Key Parsing Helpers

The package provides helper functions to parse PEM-encoded keys:

```go
// Parse RSA public key from PEM
pemData, _ := os.ReadFile("rsa_public.pem")
rsaKey, err := jwt.ParseRSAPublicKeyFromPEM(pemData)

// Parse ECDSA public key from PEM
pemData, _ := os.ReadFile("ecdsa_public.pem")
ecdsaKey, err := jwt.ParseECDSAPublicKeyFromPEM(pemData)
```

Supported formats:
- RSA: `PUBLIC KEY` (PKIX) and `RSA PUBLIC KEY` (PKCS#1)
- ECDSA: `PUBLIC KEY` (PKIX) and `EC PUBLIC KEY`

## Error Types

The middleware returns descriptive errors for different validation failures:

| Error | Cause |
|-------|-------|
| `ErrMissingAuthHeader` | Authorization header is missing |
| `ErrInvalidAuthHeader` | Authorization header format is invalid (not "Bearer <token>") |
| `ErrInvalidToken` | JWT token cannot be parsed |
| `ErrTokenExpired` | Token's `exp` claim is in the past |
| `ErrTokenNotYetValid` | Token's `nbf` claim is in the future |
| `ErrInvalidSignature` | JWT signature verification failed |
| `ErrInvalidIssuer` | Token's `iss` claim doesn't match expected issuer |
| `ErrInvalidAudience` | Token's `aud` claim doesn't contain expected audience |
| `ErrInvalidIssuedAt` | Token's `iat` claim is in the future |
| `ErrMissingRequiredClaim` | A required claim is missing |
| `ErrUnsupportedAlgorithm` | JWT algorithm is not in allowed list |
| `ErrJWKSFetchFailed` | Failed to fetch JWKS from endpoint |
| `ErrNoKeySourceConfigured` | No key source configured (Secret, PublicKey, or JWKS) |
| `ErrMultipleKeySourcesConfigured` | Multiple key sources configured (mutually exclusive) |

## Troubleshooting

### Issue: "missing authorization header"

**Cause:** The request doesn't include an `Authorization` header.

**Solutions:**
1. Ensure clients include the header: `Authorization: Bearer <token>`
2. Use `jwt.WithOptional(true)` if some endpoints should work without tokens
3. Use `jwt.WithSkipPaths(...)` to exclude specific services from validation

### Issue: "invalid authorization header format"

**Cause:** The `Authorization` header doesn't follow the `Bearer <token>` format.

**Solutions:**
1. Ensure the header value starts with `Bearer ` (case-insensitive)
2. Check for extra spaces or incorrect formatting
3. Verify the token is not empty after the prefix

### Issue: "token expired"

**Cause:** The token's `exp` claim is in the past.

**Solutions:**
1. Refresh the token on the client side
2. Increase `jwt.WithClockSkew(duration)` if time drift is an issue
3. Check server and client clocks are synchronized

### Issue: "invalid token signature"

**Cause:** Signature verification failed.

**Solutions:**
1. Verify you're using the correct key (secret or public key)
2. Check the token was signed with a compatible algorithm
3. Ensure the key hasn't been rotated (for JWKS, the middleware auto-refreshes)
4. Verify the token hasn't been tampered with

### Issue: "unsupported algorithm"

**Cause:** The token's algorithm is not in the `AllowedAlgorithms` list.

**Solutions:**
1. Add the algorithm to `jwt.WithAllowedAlgorithms(...)`
2. If not specified, ensure your key type matches the token algorithm (e.g., RSA key for RS256)
3. Check for algorithm confusion attacks (e.g., RS256 token with HS256 validation)

### Issue: "failed to fetch JWKS"

**Cause:** Cannot fetch keys from the JWKS endpoint.

**Solutions:**
1. Verify the JWKS endpoint URL is correct and accessible
2. Check network connectivity and firewall rules
3. Ensure the endpoint returns valid JWKS JSON
4. Increase timeout with proper JWKS configuration
5. Check logs for detailed error messages

### Issue: "invalid issuer" or "invalid audience"

**Cause:** Token's `iss` or `aud` claims don't match expected values.

**Solutions:**
1. Verify `WithExpectedIssuer()` matches the token's `iss` claim exactly
2. Ensure at least one `WithExpectedAudience()` value matches the token's `aud`
3. Check for trailing slashes in issuer URLs (must match exactly)
4. Decode the token at jwt.io to inspect claims

### Issue: "missing required claim"

**Cause:** A claim specified in `WithRequiredClaims()` is missing or empty.

**Solutions:**
1. Ensure the token includes all required claims
2. Check the token issuer is configured to include these claims
3. Verify claim names are spelled correctly (case-sensitive)

### Issue: High latency or performance issues

**Cause:** JWKS fetching on every request or inefficient validation.

**Solutions:**
1. Ensure `WithJWKSCacheTTL()` is set to a reasonable value (default: 1 hour)
2. Enable background refresh with `WithJWKSRefreshInterval()` (default: 50 minutes)
3. Increase cache TTL for stable key sets
4. Monitor JWKS endpoint response times
5. Use static keys (Secret or PublicKey) if key rotation isn't needed

### Issue: Panics with "jwt: claims not found in context"

**Cause:** `MustClaimsFromContext()` called when claims aren't present.

**Solutions:**
1. Use `ClaimsFromContext()` instead and check the boolean return value
2. Ensure the service path isn't in `SkipPaths`
3. Check that `Optional: true` isn't enabled when you expect authentication
4. Verify JWT validation succeeded (check logs for errors)

## Advanced Configuration

### Custom Header and Prefix

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithHeaderKey("X-API-Token"),       // Custom header name
    jwt.WithTokenPrefix("Token "),          // Custom prefix
)
// Expects: X-API-Token: Token eyJhbGc...
```

### Strict Time Validation (No Clock Skew)

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithClockSkew(0), // No tolerance for clock drift
)
```

### Algorithm Whitelisting (Security Best Practice)

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithAllowedAlgorithms("HS256"), // Only allow HS256
)
```

### Multiple Audiences

```go
jwtMw, err := jwt.New(
    jwt.WithSecret([]byte("secret")),
    jwt.WithExpectedAudience("api-v1", "api-v2", "admin-api"),
    // Token's aud must contain at least one of these
)
```

## Performance

- **Static Key Validation:** <10ms (p95)
- **JWKS Cached Validation:** <20ms (p95)
- **Test Coverage:** 89.8%
- **Thread Safety:** All components are thread-safe and race-free
- **Production Ready:** Tested with `go test -race` and stress tests

## Dependencies

- `github.com/golang-jwt/jwt/v5` - JWT parsing and validation
- `github.com/lestrrat-go/jwx/v2` - JWKS parsing

## License

See the main Mono framework repository for license information.

## Contributing

Contributions are welcome! Please follow the Mono framework contribution guidelines.

## Support

For issues and questions:
- GitHub Issues: https://github.com/go-monolith/mono/issues
- Documentation: https://github.com/go-monolith/mono

## Related

- [Mono Framework](https://github.com/go-monolith/mono)
- [OpenTelemetry Middleware](../otel/)
