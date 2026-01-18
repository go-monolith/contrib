# JWT Middleware Requirements (Mono Framework)

## Introduction

The **JWT middleware** is a contribution module for the Mono Framework that validates JWT tokens in NATS message headers. It implements the `mono.MiddlewareModule` interface and wraps all Mono handler types to enforce authentication before handler execution.

This middleware extracts JWT tokens from message headers, validates signatures and claims, and enriches the request context with validated claims for use in downstream handlers.

**Reference:** This expands on the existing `REQUIREMENTS.md` in the parent directory.

---

## Functional Requirements

### FR1: Message Header Extraction

**User Story:** As a Mono framework developer, I want to extract JWT tokens from NATS message headers, so that I can validate authentication before processing messages.

**Acceptance Criteria:**

1. WHEN a Mono message is received with `Authorization` header THEN the middleware SHALL extract the header value
2. WHEN the header key is in any case (Authorization, authorization, AUTHORIZATION) THEN the middleware SHALL find it (case-insensitive)
3. WHEN the header value follows `Bearer <token>` format THEN the middleware SHALL extract the token part
4. WHEN the header value has variations like `bearer <token>` or `BEARER <token>` THEN the middleware SHALL handle case-insensitively
5. WHEN the `Authorization` header is missing THEN the middleware SHALL return error "missing authorization header" (unless `Optional: true`)
6. WHEN the header format is invalid (not "Bearer <token>") THEN the middleware SHALL return error "invalid authorization header format"
7. WHEN `Optional: true` is configured AND header is missing THEN the middleware SHALL allow the request to proceed without claims

### FR2: JWT Signature Validation

**User Story:** As a security-conscious developer, I want JWT signatures to be cryptographically verified, so that I can trust the token's authenticity.

**Acceptance Criteria:**

1. WHEN using static `Secret` (HMAC) THEN the middleware SHALL verify signature using configured secret key
2. WHEN using static `PublicKey` (RSA/ECDSA) THEN the middleware SHALL verify signature using configured public key
3. WHEN using JWKS endpoint THEN the middleware SHALL fetch the appropriate key based on `kid` header claim
4. WHEN signature verification succeeds THEN the middleware SHALL proceed to claims validation
5. WHEN signature verification fails THEN the middleware SHALL return error "invalid token signature"
6. WHEN algorithm in JWT header is not in configured `AllowedAlgorithms` THEN the middleware SHALL return error "unsupported algorithm"
7. WHEN no `AllowedAlgorithms` is configured THEN the middleware SHALL accept algorithms matching the key type (HMAC for Secret, RSA for RSA keys, etc.)

### FR3: Standard Claims Validation

**User Story:** As a developer, I want standard JWT claims to be validated, so that expired or not-yet-valid tokens are rejected.

**Acceptance Criteria:**

1. WHEN the `exp` (expiration) claim is present AND in the past THEN the middleware SHALL return error "token expired"
2. WHEN the `nbf` (not before) claim is present AND in the future THEN the middleware SHALL return error "token not yet valid"
3. WHEN validating time-based claims THEN the middleware SHALL apply configured `ClockSkew` (default: 1 minute) to account for clock drift
4. WHEN the `iat` (issued at) claim is present AND is in the future THEN the middleware SHALL return error "invalid issued at time"
5. WHEN time claims are missing THEN the middleware SHALL continue validation (these claims are optional in JWT spec)
6. WHEN `ClockSkew` is configured to 0 THEN the middleware SHALL perform strict time validation with no tolerance

### FR4: Issuer and Audience Validation

**User Story:** As a developer, I want to validate issuer and audience claims, so that tokens from unauthorized sources are rejected.

**Acceptance Criteria:**

1. WHEN `ExpectedIssuer` is configured AND `iss` claim matches THEN validation SHALL succeed
2. WHEN `ExpectedIssuer` is configured AND `iss` claim does not match THEN the middleware SHALL return error "invalid issuer"
3. WHEN `ExpectedIssuer` is NOT configured THEN the middleware SHALL skip issuer validation
4. WHEN `ExpectedAudience` is configured AND `aud` claim contains at least one matching audience THEN validation SHALL succeed
5. WHEN `ExpectedAudience` is configured AND `aud` claim does not match any expected audience THEN the middleware SHALL return error "invalid audience"
6. WHEN `ExpectedAudience` is NOT configured THEN the middleware SHALL skip audience validation
7. WHEN the `aud` claim is a string (single audience) OR array (multiple audiences) THEN the middleware SHALL handle both formats correctly

### FR5: Required Claims Validation

**User Story:** As a developer, I want to enforce presence of specific claims, so that handlers can rely on required data being available.

**Acceptance Criteria:**

1. WHEN `RequiredClaims` is configured with claim names THEN the middleware SHALL verify all specified claims are present
2. WHEN any required claim is missing THEN the middleware SHALL return error "missing required claim: <claim_name>"
3. WHEN all required claims are present THEN validation SHALL continue
4. WHEN `RequiredClaims` is empty or not configured THEN the middleware SHALL skip this validation
5. WHEN a required claim exists but has `null` or empty value THEN the middleware SHALL treat it as missing

### FR6: Context Enhancement with Claims

**User Story:** As a handler developer, I want validated claims available in the request context, so that I can access user identity and authorization data.

**Acceptance Criteria:**

1. WHEN JWT validation succeeds THEN the middleware SHALL extract all claims from the token payload
2. WHEN claims are extracted THEN the middleware SHALL store them in `context.Context` using a type-safe context key
3. WHEN claims are stored THEN downstream handlers SHALL be able to retrieve them using `ClaimsFromContext(ctx)`
4. WHEN the `sub` (subject) claim exists THEN it SHALL be accessible via `SubjectFromContext(ctx)` helper
5. WHEN claims are retrieved from context THEN they SHALL be of type `jwt.MapClaims` (map[string]interface{})
6. WHEN validation fails THEN claims SHALL NOT be added to context and the original handler SHALL NOT be called

### FR7: JWKS Endpoint Support

**User Story:** As a platform operator, I want to fetch public keys dynamically from a JWKS endpoint, so that I can support key rotation without redeploying my application.

**Acceptance Criteria:**

1. WHEN `JWKSEndpoint` is configured THEN the middleware SHALL fetch the JWKS document on startup
2. WHEN fetching JWKS on startup fails THEN the middleware SHALL return error from `Start()` method and prevent application startup
3. WHEN a JWT has a `kid` (key ID) header THEN the middleware SHALL look up the corresponding key in the JWKS cache
4. WHEN the key is found in cache THEN the middleware SHALL use it for signature verification
5. WHEN the key is NOT found in cache THEN the middleware SHALL refresh the JWKS and retry verification once
6. WHEN key is still not found after refresh THEN the middleware SHALL return error "key not found: kid=<kid>"
7. WHEN JWT does not have a `kid` header AND JWKS contains only one key THEN the middleware SHALL use that key

### FR8: JWKS Caching Strategy

**User Story:** As a platform operator, I want JWKS keys to be cached efficiently, so that my application doesn't make excessive HTTP requests.

**Acceptance Criteria:**

1. WHEN JWKS is fetched successfully THEN the middleware SHALL cache all keys indexed by `kid`
2. WHEN the cache is older than `JWKSCacheTTL` (default: 1 hour) THEN the middleware SHALL refresh on the next validation attempt
3. WHEN `JWKSRefreshInterval` is configured THEN the middleware SHALL start a background goroutine to refresh proactively
4. WHEN background refresh succeeds THEN the cache SHALL be updated atomically without blocking message processing
5. WHEN background refresh fails THEN the middleware SHALL log a warning and continue using stale cache
6. WHEN signature verification fails with a cached key THEN the middleware SHALL immediately refresh JWKS and retry once
7. WHEN multiple concurrent messages trigger JWKS refresh THEN only one HTTP request SHALL be made (deduplication via mutex)

### FR9: Handler Wrapping for All Handler Types

**User Story:** As a Mono framework developer, I want JWT validation to apply to all handler types, so that authentication is consistently enforced.

**Acceptance Criteria:**

1. WHEN the middleware wraps a `RequestReplyHandler` THEN JWT validation SHALL occur before the handler is called
2. WHEN the middleware wraps a `QueueGroupHandler` THEN JWT validation SHALL occur before the handler is called
3. WHEN the middleware wraps a `StreamConsumerHandler` THEN JWT validation SHALL occur before the handler is called
4. WHEN the middleware wraps an `EventConsumerHandler` THEN JWT validation SHALL occur before the handler is called
5. WHEN the middleware wraps an `EventStreamConsumerHandler` THEN JWT validation SHALL occur before the handler is called
6. WHEN validation succeeds for any handler type THEN the enhanced context with claims SHALL be passed to the original handler
7. WHEN validation fails for any handler type THEN the original handler SHALL NOT be called and an error SHALL be returned

### FR10: Skip Paths Configuration

**User Story:** As a developer, I want to exclude certain service paths from JWT validation, so that public endpoints can be accessed without tokens.

**Acceptance Criteria:**

1. WHEN `SkipPaths` is configured with service names THEN the middleware SHALL not validate JWT for those services
2. WHEN a service name matches a skip path pattern THEN the middleware SHALL call the original handler without validation
3. WHEN a service name does NOT match any skip path THEN the middleware SHALL perform JWT validation
4. WHEN skip path matching is used THEN it SHALL support exact matches (e.g., "module.service.method")
5. WHEN `SkipPaths` is empty or not configured THEN the middleware SHALL validate all requests

### FR11: Configuration and Initialization

**User Story:** As a developer, I want to configure the JWT middleware using functional options, so that I have a clean and flexible API.

**Acceptance Criteria:**

1. WHEN initializing with `jwt.New()` THEN it SHALL accept variadic functional options
2. WHEN neither `WithSecret`, `WithPublicKey`, nor `WithJWKSEndpoint` is provided THEN `New()` SHALL return error "no key source configured"
3. WHEN both `WithSecret` and `WithPublicKey` are provided THEN `New()` SHALL return error "multiple key sources configured" (mutually exclusive)
4. WHEN `WithJWKSEndpoint` is provided with static keys THEN JWKS SHALL take precedence and static keys SHALL be ignored
5. WHEN no `JWKSCacheTTL` is specified THEN it SHALL default to 1 hour
6. WHEN no `ClockSkew` is specified THEN it SHALL default to 1 minute
7. WHEN no `HeaderKey` is specified THEN it SHALL default to "authorization"
8. WHEN no `TokenPrefix` is specified THEN it SHALL default to "Bearer "

---

## Non-Functional Requirements

### NFR1: Mono Framework Compatibility

**User Story:** As a Mono framework developer, I want the JWT middleware to integrate seamlessly, so that I can use it like any other middleware module.

**Acceptance Criteria:**

1. WHEN the middleware is created THEN it SHALL implement the `mono.MiddlewareModule` interface
2. WHEN registered with `app.Register(jwtMw)` THEN it SHALL integrate with Mono's lifecycle management
3. WHEN the middleware implements `OnServiceRegistration` THEN it SHALL wrap handlers as shown in `v1/otel/` reference
4. WHEN the application starts THEN the middleware's `Start(ctx)` method SHALL be called for initialization
5. WHEN the application stops THEN the middleware's `Stop(ctx)` method SHALL be called for cleanup
6. WHEN using the middleware THEN it SHALL work with all Mono plugins (kv-jetstream, fs-jetstream, etc.)

### NFR2: Security Best Practices

**User Story:** As a security engineer, I want the middleware to follow cryptographic best practices, so that my application is protected against known attacks.

**Acceptance Criteria:**

1. WHEN verifying signatures THEN the middleware SHALL use constant-time comparison where applicable (provided by crypto libraries)
2. WHEN logging errors THEN the middleware SHALL NOT log full token values (only redacted versions like "Bearer ***")
3. WHEN logging errors THEN the middleware SHALL NOT expose key material or algorithm details
4. WHEN handling keys THEN the middleware SHALL store them securely in memory with no unnecessary copies
5. WHEN an algorithm confusion attack is attempted THEN the middleware SHALL reject tokens with unexpected algorithms
6. WHEN the middleware has dependencies THEN they SHALL have no known CVEs (check with `go list -m all | nancy sleuth`)

### NFR3: Performance and Concurrency

**User Story:** As a platform operator, I want the JWT middleware to handle high message throughput efficiently, so that it doesn't become a bottleneck.

**Acceptance Criteria:**

1. WHEN validating JWTs with static keys THEN validation SHALL complete in <10ms (p95)
2. WHEN validating JWTs with cached JWKS keys THEN validation SHALL complete in <20ms (p95)
3. WHEN multiple concurrent messages arrive THEN the middleware SHALL handle them safely with no race conditions
4. WHEN JWKS cache is being refreshed THEN concurrent validations SHALL not be blocked (use sync.Map or RWMutex)
5. WHEN processing 1000 messages/second THEN CPU usage for JWT validation SHALL be <10% on a typical server
6. WHEN running with `-race` flag THEN no race conditions SHALL be detected

### NFR4: Testability and Coverage

**User Story:** As a developer, I want comprehensive test coverage, so that I can trust the middleware's reliability in production.

**Acceptance Criteria:**

1. WHEN unit tests are run THEN they SHALL achieve >80% code coverage for validation logic
2. WHEN security test cases are run THEN they SHALL include tests for all threat scenarios (expired tokens, invalid signatures, algorithm confusion, etc.)
3. WHEN JWKS mode is tested THEN it SHALL use a mocked HTTP server to simulate JWKS endpoint behavior
4. WHEN integration tests are run THEN they SHALL test handler wrapping for all 5 handler types
5. WHEN running benchmarks THEN they SHALL measure validation latency for static keys and JWKS cache hits
6. WHEN testing concurrent access THEN tests SHALL verify thread-safety of JWKS cache

### NFR5: Documentation and Examples

**User Story:** As a new developer, I want clear documentation and examples, so that I can integrate the middleware quickly.

**Acceptance Criteria:**

1. WHEN reading the README.md THEN it SHALL include:
   - Overview of features
   - Installation instructions
   - Quick start guide (static key and JWKS modes)
   - Configuration options reference
   - Usage examples for all 5 handler types
   - Troubleshooting guide
2. WHEN reading GoDoc THEN all exported types and functions SHALL have documentation comments
3. WHEN looking for examples THEN the repository SHALL include working sample applications
4. WHEN reviewing the code THEN complex logic SHALL have inline comments explaining the "why"

### NFR6: Logging and Observability

**User Story:** As a platform operator, I want structured logging for authentication events, so that I can monitor and debug JWT validation in production.

**Acceptance Criteria:**

1. WHEN the middleware logs THEN it SHALL use the `*slog.Logger` provided by Mono framework via `Logger()` method
2. WHEN validation fails THEN it SHALL log at WARN level with fields: `module`, `service`, `error`, `reason`, `timestamp`
3. WHEN validation succeeds THEN it MAY log at DEBUG level with fields: `module`, `service`, `subject`, `timestamp`
4. WHEN JWKS refresh occurs THEN it SHALL log at INFO level with fields: `endpoint`, `keys_count`, `timestamp`
5. WHEN JWKS refresh fails THEN it SHALL log at ERROR level with full error context
6. WHEN log level is set to INFO or higher THEN DEBUG validation logs SHALL be suppressed

### NFR7: Resource Management

**User Story:** As a platform operator, I want the middleware to properly manage resources, so that there are no memory leaks or goroutine leaks.

**Acceptance Criteria:**

1. WHEN the middleware starts THEN it SHALL initialize JWKS background refresh goroutine (if configured)
2. WHEN the middleware stops THEN it SHALL cancel background goroutines using context cancellation
3. WHEN the middleware stops THEN it SHALL wait for goroutines to exit before returning from `Stop()`
4. WHEN JWKS cache is cleared THEN it SHALL release all stored keys
5. WHEN running for extended periods THEN memory usage SHALL remain stable (no leaks)
6. WHEN stress testing with 100k messages THEN no goroutines SHALL leak (check with runtime.NumGoroutine())

---

## Integration Requirements

### IR1: Reference Implementation Pattern

**User Story:** As a contributor, I want to follow the established middleware pattern, so that my code is consistent with other contrib modules.

**Acceptance Criteria:**

1. WHEN implementing the middleware THEN it SHALL follow the patterns from `v1/otel/` reference implementation
2. WHEN wrapping handlers THEN it SHALL use the same approach as OpenTelemetry middleware
3. WHEN managing lifecycle THEN it SHALL use the same `Start()`/`Stop()` pattern
4. WHEN storing context data THEN it SHALL use unexported struct types for context keys (same as otel)

### IR2: Mono Framework Message Structure

**User Story:** As a Mono framework developer, I want the middleware to work with Mono message structure, so that it integrates seamlessly.

**Acceptance Criteria:**

1. WHEN extracting headers THEN it SHALL use `msg.Header` from `*types.Msg`
2. WHEN accessing header values THEN it SHALL handle case-insensitive lookup (Mono headers are case-insensitive)
3. WHEN the message context is available THEN it SHALL use it for cancellation and timeouts
4. WHEN returning errors THEN it SHALL use appropriate Mono error types (if any) or standard Go errors

---

*Last Updated: January 2026*
*Version: 1.0 (MVP Scope)*
