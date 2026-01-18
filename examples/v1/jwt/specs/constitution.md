# JWT Middleware Example - Constitution

## Overview

This constitution establishes non-negotiable principles and constraints for the **JWT middleware example application** for the Mono Framework. This example demonstrates how to use the JWT middleware (`/home/leo/Projects/myspec/mono_framework/contrib/v1/jwt`) to build a secure REST API with three different JWT authentication strategies.

## Core Principles

Non-negotiable principles that guide all decisions:

- **Educational First**: Code must be clear, well-documented, and serve as a learning resource for Mono framework users

- **Production Patterns**: While this is an example, it must demonstrate production-ready patterns (error handling, testing, graceful shutdown)

- **Security Awareness**: Demonstrate security best practices while providing clear warnings about development vs production configurations

- **Framework Alignment**: Follow established patterns from `mono-recipes` examples for module structure, dependency injection, and lifecycle management

- **Simplicity Over Features**: Prefer simple, focused examples over feature-complete applications. The goal is learning, not completeness.

## Technology Constraints

Required or prohibited technologies:

- **MUST use**: Go 1.21+ - Required for Mono framework
- **MUST use**: [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Mono framework
- **MUST use**: JWT middleware from `/home/leo/Projects/myspec/mono_framework/contrib/v1/jwt`
- **MUST use**: [github.com/gofiber/fiber/v2](https://github.com/gofiber/fiber) - HTTP framework (most common in mono-recipes)
- **MUST use**: [github.com/golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) - For token generation helpers
- **MUST use**: [github.com/google/uuid](https://github.com/google/uuid) - For ID generation
- **MUST use**: [github.com/gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - For signal handling
- **MUST NOT use**: Real databases - Use mock in-memory repository only
- **MUST NOT use**: External dependencies beyond those listed above

## Architecture Constraints

Architectural requirements and patterns that must be followed:

- **Hexagonal Architecture**: Demonstrate ports & adapters pattern as shown in `mono-recipes/hexagonal-architecture`

- **Module Separation**:
  - **Project Module**: Business logic, provides CRUD services via `ServiceProviderModule`
  - **HTTP Module**: REST API, depends on project module via adapter pattern
  - **JWT Middleware**: Wraps all service handlers, validates tokens before execution

- **No cmd/ Consolidation**: Each JWT strategy (static, secret provider, JWKS) must have its own `cmd/<strategy>/main.go` file

- **Port Adapter Pattern**: HTTP module must use adapter to call project services (never direct service calls)

- **Service Communication**: All inter-module communication via Mono's `RequestReplyService` pattern

- **Mock Repository**: Use thread-safe in-memory map for data storage - NO external databases

## JWT Strategy Constraints

Requirements for the three JWT authentication strategies:

### 1. Static Secret (`cmd/static`)
- Single shared HMAC secret (HS256)
- Simple token generation helper
- Development/testing use case
- Port 3000

### 2. Secret Provider (`cmd/secret`)
- Per-tenant/issuer secret lookup
- Map-based secret store
- Multi-tenant use case
- Port 3001

### 3. JWKS (`cmd/jwks`)
- RSA public key validation (RS256)
- Mock JWKS server for demo
- OAuth2 provider integration use case
- Port 3002

**Constraint**: All three strategies must share the same modules (project, http) - only the `main.go` and configuration differ

## Security Constraints

Security-related constraints and demonstration requirements:

- **Token Validation**: All examples must demonstrate proper JWT validation via middleware

- **Authorization**: All CRUD operations must check project ownership (users can only access their own projects)

- **Security Warnings**: Examples using default secrets must display clear warnings at startup

- **No Production Secrets**: All default secrets must be clearly marked as insecure for production

- **Error Messages**: Authentication errors must be informative but not leak sensitive information

- **HTTPS Note**: README must note that production should use HTTPS (examples use HTTP for simplicity)

## Testing Constraints

Mandatory testing approaches:

- **Test Coverage**:
  - Repository: 100%
  - Service handlers: 90%+
  - HTTP handlers: 85%+

- **Test Layers**:
  - **Unit Tests**: Repository, service handlers, HTTP handlers (with mocks)
  - **Integration Tests**: Each JWT strategy with real middleware
  - **No E2E Tests**: Integration tests are sufficient for examples

- **Test Data**: Use consistent test data across all tests (user-123, proj-456, etc.)

- **Mock Context**: Provide helper to create context with JWT claims for service tests

- **Test Organization**: Follow Go conventions - `*_test.go` files in same package

## Code Organization Constraints

Directory structure requirements:

```
/home/leo/Projects/myspec/mono_framework/contrib/examples/v1/jwt/
├── docs/specs/              # Specification files (this file)
├── domain/project/          # Domain models (entity, types)
├── modules/
│   ├── project/            # Project module (ServiceProviderModule)
│   └── http/               # HTTP module (DependentModule)
├── cmd/
│   ├── static/             # Static secret binary + token generator
│   ├── secret/             # Secret provider binary + token generator
│   └── jwks/               # JWKS binary + token generator + mock server
├── Makefile                # Build automation
├── go.mod                  # Go module
└── README.md               # Usage documentation
```

**Constraints**:
- NO `internal/` directory - this is an example, keep it flat
- NO `pkg/` directory - all code in `domain/` or `modules/`
- NO nested modules - single `go.mod` at root

## Module Design Constraints

Requirements for Mono modules:

### Project Module
- **Type**: `mono.Module` + `mono.ServiceProviderModule`
- **Name**: "project"
- **Services**: create, get, list, update, delete (5 services)
- **Subject Pattern**: `services.project.<action>`
- **Dependencies**: None
- **Repository**: Mock in-memory with `sync.RWMutex`

### HTTP Module
- **Type**: `mono.Module` + `mono.DependentModule` + `mono.HealthCheckableModule`
- **Name**: "http"
- **Dependencies**: ["project"]
- **Port**: Configurable via constructor
- **Lifecycle**: Start server in goroutine, graceful shutdown in Stop()
- **Adapter**: Use project adapter for service calls

## REST API Constraints

HTTP endpoint requirements:

- **Endpoint Pattern**: `/api/v1/<resource>`
- **HTTP Methods**: Standard REST (POST, GET, PUT, DELETE)
- **Content Type**: `application/json` only
- **Authentication**: All `/api/v1/projects` endpoints require `Authorization: Bearer <token>`
- **Health Check**: `GET /health` must not require authentication
- **Error Format**: `{"error": "...", "message": "..."}`
- **Success Format**: Return entity or list directly (no wrapper objects)

## Build and Development Constraints

Build system requirements:

- **Makefile Required**: Provide comprehensive Makefile with:
  - Build targets for each binary
  - Run targets for each example
  - Token generation targets
  - Test targets (unit, integration, coverage)
  - Code quality targets (fmt, lint, tidy)

- **Binary Names**:
  - `bin/jwt-example-static`
  - `bin/jwt-example-secret`
  - `bin/jwt-example-jwks`

- **Build Flags**: Use `-trimpath -ldflags="-s -w"` for smaller binaries

- **Color Output**: Makefile should use ANSI colors for better UX

## Documentation Constraints

Documentation requirements:

- **README.md**: Must include:
  - Project overview
  - Prerequisites
  - Quick start for each JWT strategy
  - Example curl commands
  - When to use each strategy

- **Code Comments**: All exported types, functions, and methods must have godoc comments

- **Inline Comments**: Complex logic must have explanatory comments

- **Token Generation**: Each strategy must have working token generation helper with usage instructions

## Error Handling Constraints

Error handling requirements:

- **Service Layer**: Return errors with context (`fmt.Errorf("failed to X: %w", err)`)

- **HTTP Layer**: Map service errors to appropriate HTTP status codes:
  - 400 Bad Request: Validation errors
  - 401 Unauthorized: Missing/invalid JWT
  - 403 Forbidden: Authorization failures (not owner)
  - 404 Not Found: Entity not found
  - 500 Internal Server Error: Unexpected errors

- **Error Classification**: HTTP module must have helper functions (`isUnauthorizedError`, `isForbiddenError`, etc.)

- **Logging**: Log all errors with context (user ID, project ID, action)

## Performance Constraints

Performance requirements (for educational purposes):

- **Startup Time**: All examples should start in <2 seconds

- **Request Latency**: REST API requests should complete in <100ms (excluding JWT validation)

- **Concurrent Safety**: Repository must be thread-safe (use `sync.RWMutex`)

- **No Load Testing**: This is an example, performance benchmarks are not required

## Coding Standards

Required coding practices:

- **Idiomatic Go**: Follow Go best practices, Effective Go guidelines

- **gofmt**: All code must be formatted with `gofmt`

- **No Panic**: Never use `panic()` in production code (only in tests or startup validation)

- **Error Checking**: Always check errors, never use `_` to discard errors

- **Context Propagation**: Always pass `context.Context` as first parameter

- **Graceful Shutdown**: All examples must handle SIGINT/SIGTERM gracefully

## Mono Framework Integration Constraints

Specific requirements for Mono framework integration:

- **Module Registration Order**:
  1. JWT middleware (wraps handlers)
  2. Project module (provides services)
  3. HTTP module (depends on project)

- **Lifecycle Methods**:
  - `Start(ctx)`: Initialize module, start servers
  - `Stop(ctx)`: Clean up resources, stop servers
  - `Name()`: Return module name (lowercase, kebab-case)

- **Dependency Injection**: Use `SetDependencyServiceContainer()` to receive dependencies

- **Service Registration**: Use `helper.RegisterTypedRequestReplyService()` for type safety

## Token Generation Constraints

Requirements for token generation helpers:

- **Separate Files**: `cmd/<strategy>/generate_token.go` for each strategy

- **Standalone Execution**: Must work with `go run cmd/<strategy>/generate_token.go`

- **Environment Variables**: Support customization via env vars (USER_ID, EMAIL, TENANT, etc.)

- **JSON Output**: Support `--json` flag for scripting

- **Human-Readable Output**: Default output should be user-friendly with usage instructions

## Reference Examples Constraints

Examples that must be referenced and followed:

- **Module Pattern**: `mono-recipes/jwt-auth-demo` - JWT middleware usage
- **CRUD Pattern**: `mono-recipes/gorm-sqlite-demo` - Service provider pattern
- **HTTP Integration**: `mono-recipes/background-jobs-demo` - Fiber integration
- **Graceful Shutdown**: `mono-recipes/graceful-shutdown-demo` - Signal handling

**Constraint**: Follow these patterns exactly unless there's a compelling reason to differ (document any deviations)

## Additional Constraints

Any other constraints specific to this example:

- **No User Management**: Do NOT implement user registration/login - JWT sub claim is the user ID

- **No Password Hashing**: This example focuses on JWT, not authentication flows

- **No Database Migrations**: Mock repository means no migrations needed

- **No Configuration Files**: Use environment variables only (no YAML/JSON config files)

- **Single Responsibility**: Each module should do one thing well

- **README Examples**: All README examples must be tested and work as documented

- **Cross-Platform**: Makefile should work on Linux and macOS (document Windows limitations)

---

*Last Updated: January 2026*
*Version: 1.0*
