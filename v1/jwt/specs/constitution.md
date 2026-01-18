# JWT Middleware Constitution (Mono Framework)

## Overview

This constitution establishes non-negotiable principles and constraints for the **JWT middleware** contribution module for the Mono Framework. This middleware validates JWT tokens in NATS message headers and implements the `mono.MiddlewareModule` interface to wrap Mono framework handlers.

## Core Principles

Non-negotiable principles that guide all decisions:

- **Security First**: JWT validation must be cryptographically sound. All tokens in message headers must be properly verified before handler execution.

- **Zero Trust**: Never trust incoming message headers. Always validate signature, expiration, and required claims.

- **Fail Secure**: When validation fails, prevent handler execution. Invalid tokens should never reach business logic.

- **Mono Framework Integration**: Follow established patterns from `v1/otel/` reference implementation for middleware lifecycle and handler wrapping.

## Technology Constraints

Required or prohibited technologies:

- **MUST use**: Go programming language - Aligned with Mono Framework
- **MUST use**: [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Mono framework for middleware interface
- **MUST use**: [github.com/golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) - JWT parsing and validation library
- **MUST support**: Multiple signature algorithms:
  - HMAC: HS256, HS384, HS512
  - RSA: RS256, RS384, RS512
  - ECDSA: ES256, ES384, ES512
- **MUST NOT use**: External HTTP client libraries for JWKS (use stdlib `net/http` only)

## Architecture Constraints

Architectural requirements and patterns that must be followed:

- **Mono MiddlewareModule Pattern**: Implement the `mono.MiddlewareModule` interface with all required methods (`Name()`, `Start()`, `Stop()`, `OnServiceRegistration()`, etc.)

- **Handler Wrapping**: Wrap ALL Mono handler types:
  - `types.RequestReplyHandler`
  - `types.QueueGroupHandler`
  - `types.StreamConsumerHandler`
  - `types.EventConsumerHandler`
  - `types.EventStreamConsumerHandler`

- **Message Header Extraction**: Extract JWT from `mono.Message.Header["authorization"]` with support for case-insensitive header keys and `Bearer <token>` format

- **Context Enhancement**: Store validated claims in `context.Context` using type-safe context keys for retrieval in downstream handlers

- **Dual Mode Support**: Support both static key mode and JWKS mode with clear configuration separation:
  - **Static Key Mode**: Pre-configured `Secret` (HMAC) or `PublicKey` (RSA/ECDSA)
  - **JWKS Mode**: Dynamic key fetching from JWKS endpoint with caching

- **JWKS Caching Strategy**:
  - Cache keys for 1 hour (default, configurable)
  - Background refresh before TTL expiration
  - Graceful key rotation handling via `kid` matching
  - Thread-safe cache operations

## Security Constraints

Security-related constraints and compliance requirements:

- **Signature Verification Required**: Every JWT must have its signature verified against the configured key source before handler execution

- **Standard Claims Validation**: Validate `exp`, `nbf`, and `iat` claims with configurable clock skew (default: 1 minute)

- **Optional Claims Validation**: Support validation of `iss` (issuer) and `aud` (audience) when configured

- **Required Claims Enforcement**: Allow configuration of additional required claims beyond standard claims

- **Algorithm Whitelist**: Support configurable algorithm whitelist to prevent algorithm confusion attacks

- **No Token Logging**: Never log full JWT tokens or signing keys in production (only redacted/masked values)

## Configuration Constraints

Configuration-related requirements:

- **Functional Options Pattern**: Use functional options (`WithSecret()`, `WithPublicKey()`, `WithJWKSEndpoint()`, etc.) for clean configuration API

- **Mutually Exclusive Key Sources**: Enforce that only ONE key source is configured (Secret XOR PublicKey XOR JWKSEndpoint)

- **Default Values**: Provide sensible defaults:
  - `JWKSCacheTTL`: 1 hour
  - `JWKSRefreshInterval`: 50 minutes (before TTL)
  - `ClockSkew`: 1 minute
  - `HeaderKey`: "authorization"
  - `TokenPrefix`: "Bearer "

- **Environment-Based Configuration**: Support loading configuration from environment variables for production deployments

- **Optional Validation**: Support `Optional: true` config to allow requests without tokens (for mixed auth scenarios)

## Error Handling Constraints

Error handling requirements:

- **Clear Error Messages**: Return descriptive errors for all validation failures:
  - Missing Authorization header
  - Invalid token format
  - Expired token
  - Invalid signature
  - Unsupported algorithm
  - Missing required claims

- **Prevent Handler Execution**: On validation failure, return error immediately - do NOT call the original handler

- **Structured Logging**: Log all validation failures with context (service name, module name, error reason, timestamp)

- **No Sensitive Info in Errors**: Error messages must NOT expose key material, algorithm details, or full token values

## Testing Constraints

Mandatory testing approaches:

- **Unit Tests Required**: All validation logic, JWKS fetching, caching, and context helpers must have unit tests

- **Coverage Target**: >80% code coverage for core validation logic

- **Security Test Cases**: Must include tests for:
  - Expired tokens
  - Invalid signatures
  - Algorithm confusion attacks
  - Missing required claims
  - Malformed tokens
  - Token tampering attempts

- **JWKS Mock Testing**: Test JWKS fetching and caching with mocked HTTP server

- **Integration Tests**: Test with real Mono application to verify handler wrapping for all handler types

- **Handler Wrapping Tests**: Verify all 5 handler types are correctly wrapped and receive enhanced context

## Performance Constraints

Performance requirements:

- **Validation Latency**: JWT validation should complete in <10ms (p95) for static keys, <20ms for cached JWKS keys

- **JWKS Cache Efficiency**: JWKS cache should use sync.Map for concurrent access with minimal lock contention

- **Background Refresh**: JWKS refresh should happen in background goroutine without blocking message processing

- **Memory Efficiency**: JWKS cache should store only essential key data (public keys, not full JWKS document)

## Coding Standards

Required coding practices:

- **Idiomatic Go**: Follow Go best practices, Effective Go guidelines, and mono framework conventions

- **Error Wrapping**: Use `fmt.Errorf` with `%w` to preserve error context through the call stack

- **Structured Logging**: Use `*slog.Logger` from middleware's `Logger()` method (mono framework provides it)

- **Interface-Based Design**: Abstract JWKS fetcher and key validator as interfaces for testability and extensibility

- **Concurrent-Safe**: All shared state (JWKS cache) must be thread-safe using appropriate synchronization primitives

## Mono Framework Integration Constraints

Specific requirements for Mono framework integration:

- **Middleware Lifecycle**: Implement proper `Start()` and `Stop()` methods:
  - `Start()`: Initialize JWKS background refresh (if configured)
  - `Stop()`: Clean up background goroutines and release resources

- **OnServiceRegistration Hook**: Wrap handlers using the `OnServiceRegistration` hook as shown in `v1/otel/` reference

- **Logger Integration**: Use the logger provided by Mono framework via `Logger()` method

- **No Dependencies on Other Modules**: JWT middleware should not depend on other contrib modules (standalone)

- **Backward Compatibility**: Once v1 is released, maintain backward compatibility within v1.x versions

## Additional Constraints

Any other constraints specific to this middleware:

- **Skip Paths Support**: Allow configuration of service paths to skip validation (e.g., health checks, public endpoints)

- **Context Key Type Safety**: Use unexported struct type for context keys to prevent collisions

- **Helper Functions**: Provide convenience helpers for common claim retrieval (`SubjectFromContext`, `ClaimsFromContext`)

- **PEM Key Support**: Support loading RSA/ECDSA keys from PEM-encoded strings for ease of configuration

- **Documentation**: All exported functions and types must have godoc comments with usage examples

---

*Last Updated: January 2026*
*Version: 1.0*
