# JWT Middleware Specification (Mono Framework)

This directory contains the complete specification for the JWT middleware contribution module for the **Mono Framework** (NATS-based messaging).

## Context

This JWT middleware validates JWT tokens in **Mono framework message headers** (NATS messaging), not HTTP Authorization headers. It implements the `mono.MiddlewareModule` interface and wraps Mono handlers.

**Location:** `~/Projects/myspec/mono_framework/contrib/v1/jwt/`

**Existing file:** `REQUIREMENTS.md` (reference implementation requirements)

## Specification Files

### 1. [constitution.md](./constitution.md)
Non-negotiable principles and constraints for Mono framework JWT middleware

### 2. [requirements.md](./requirements.md)
Detailed functional and non-functional requirements (expands on existing REQUIREMENTS.md)

### 3. [design.md](./design.md)
Technical design for Mono framework middleware implementation

### 4. [tasks.md](./tasks.md)
Step-by-step implementation plan with milestones

## Key Differences from HTTP/Fiber JWT Middleware

| Aspect | This Middleware (Mono Framework) | HTTP/Fiber Middleware |
|--------|----------------------------------|----------------------|
| **Context** | NATS messaging | HTTP requests |
| **Interface** | `mono.MiddlewareModule` | `fiber.Handler` |
| **Token Source** | `mono.Message.Header["authorization"]` | HTTP `Authorization` header |
| **Handler Types** | RequestReply, QueueGroup, StreamConsumer, EventConsumer | Fiber route handlers |
| **Algorithms** | HMAC, RSA, ECDSA (multiple) | EdDSA/Ed25519 (platform-specific) |
| **Use Case** | Internal service-to-service auth | External API authentication |

## Implementation Reference

**Existing file:** `../REQUIREMENTS.md` contains the original requirements
**Reference middleware:** `v1/otel/` for Mono middleware patterns

---

*Specifications for Mono Framework JWT Middleware*
*Version: 1.0*
