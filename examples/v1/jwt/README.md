# JWT Middleware Example

A comprehensive example demonstrating three different JWT authentication strategies using the Mono Framework. This example showcases hexagonal architecture, multi-tenant authentication, and various JWT validation approaches.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [JWT Strategies](#jwt-strategies)
  - [Strategy 1: Static Secret](#strategy-1-static-secret-hs256)
  - [Strategy 2: Secret Provider](#strategy-2-secret-provider-multi-tenant)
  - [Strategy 3: JWKS](#strategy-3-jwks-rsa-public-keys)
- [Quick Start](#quick-start)
- [Manual Testing](#manual-testing)
- [Environment Variables](#environment-variables)
- [API Endpoints](#api-endpoints)
- [Testing](#testing)
- [When to Use Each Strategy](#when-to-use-each-strategy)

## Overview

This example implements a complete CRUD API for managing projects with JWT authentication. It demonstrates:

- **Hexagonal Architecture**: Clean separation between domain, services, and adapters
- **Three JWT Strategies**: Static secret, secret provider (multi-tenant), and JWKS (RSA)
- **Multi-Module Design**: Using Mono Framework's module system
- **Thread-Safe Implementation**: Concurrent-safe in-memory repository
- **Comprehensive Testing**: Unit tests and integration tests for each strategy

## Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile commands)
- curl (for manual testing)

## Project Structure

```
jwt/
├── cmd/
│   ├── static/              # Static secret strategy (port 3000)
│   ├── secret/              # Secret provider strategy (port 3001)
│   └── jwks/                # JWKS strategy (port 3002)
├── domain/
│   └── project/             # Domain models
├── modules/
│   ├── project/             # Project business logic
│   └── http/                # HTTP adapter (Fiber)
├── Makefile                 # Build and run commands
└── README.md                # This file
```

## JWT Strategies

### Strategy 1: Static Secret (HS256)

**Use Case**: Simple applications with a single shared secret.

- **Algorithm**: HS256 (HMAC with SHA-256)
- **Port**: 3000
- **Best For**: Development, internal tools, single-tenant applications

**Quick Start**:
```bash
# Start the server
make run-static

# Generate a token
make token-static

# Use the token (copy from output)
export JWT_TOKEN="eyJ..."
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:3000/api/v1/projects
```

### Strategy 2: Secret Provider (Multi-Tenant)

**Use Case**: Multi-tenant SaaS applications where each tenant has its own secret.

- **Algorithm**: HS256 (HMAC with SHA-256)
- **Port**: 3001
- **Best For**: Multi-tenant applications, B2B SaaS platforms

**Quick Start**:
```bash
# Start the server (with pre-configured tenants)
make run-secret

# Generate token for tenant-1
TENANT=tenant-1 USER_ID=alice make token-secret

# Generate token for tenant-2
TENANT=tenant-2 USER_ID=bob make token-secret

# Test tenant isolation
export TOKEN_TENANT1="eyJ..."
export TOKEN_TENANT2="eyJ..."
```

### Strategy 3: JWKS (RSA Public Keys)

**Use Case**: Enterprise applications using external identity providers (Auth0, Okta, etc.).

- **Algorithm**: RS256 (RSA with SHA-256)
- **Port**: 3002
- **Best For**: OAuth 2.0/OIDC integration, microservices, zero-trust architecture

**Quick Start**:
```bash
# Start the server (includes mock JWKS server on port 9000)
make run-jwks

# Generate an RS256 token
make token-jwks

# Use the token
export JWT_TOKEN="eyJ..."
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:3002/api/v1/projects
```

## Quick Start

### Build All Examples
```bash
make build
```

### Run a Specific Strategy
```bash
# Static secret
make run-static

# Secret provider
make run-secret

# JWKS
make run-jwks
```

### Run All Tests
```bash
make test
```

## Manual Testing

This section provides step-by-step manual testing instructions for each strategy.

### Testing Static Secret Strategy

#### 1. Start the Server
```bash
make run-static
```

Expected output:
```
============================================================
JWT Example - Static Secret Strategy
============================================================
HTTP Server:    http://localhost:3000
Health Check:   http://localhost:3000/health
API Endpoint:   http://localhost:3000/api/v1/projects
...
```

#### 2. Health Check
```bash
curl http://localhost:3000/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "http"
}
```

#### 3. Generate a Token
In a new terminal:
```bash
make token-static
```

Copy the token from the output and export it:
```bash
export JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 4. Create a Project
```bash
curl -X POST http://localhost:3000/api/v1/projects \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Project",
    "description": "Testing the static secret strategy"
  }'
```

Expected response (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My First Project",
  "description": "Testing the static secret strategy",
  "owner_id": "user-123",
  "created_at": "2026-01-18T10:30:00Z",
  "updated_at": "2026-01-18T10:30:00Z"
}
```

#### 5. List Projects
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3000/api/v1/projects
```

Expected response (200 OK):
```json
{
  "projects": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "My First Project",
      "description": "Testing the static secret strategy",
      "owner_id": "user-123",
      "created_at": "2026-01-18T10:30:00Z",
      "updated_at": "2026-01-18T10:30:00Z"
    }
  ],
  "total": 1
}
```

#### 6. Get a Specific Project
```bash
# Replace {project_id} with actual ID from create response
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3000/api/v1/projects/{project_id}
```

Expected response (200 OK): Same project object

#### 7. Update a Project
```bash
curl -X PUT http://localhost:3000/api/v1/projects/{project_id} \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Project Name",
    "description": "Updated description"
  }'
```

Expected response (200 OK): Updated project object

#### 8. Delete a Project
```bash
curl -X DELETE http://localhost:3000/api/v1/projects/{project_id} \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Expected response (200 OK): Deleted project object

#### 9. Test Unauthorized Access (No Token)
```bash
curl -X POST http://localhost:3000/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name": "Should Fail"}'
```

Expected response (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "message": "missing authorization header"
}
```

#### 10. Test Invalid Token
```bash
curl -H "Authorization: Bearer invalid-token-here" \
  http://localhost:3000/api/v1/projects
```

Expected response (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "message": "invalid token"
}
```

#### 11. Test Forbidden Access (Different User's Project)
```bash
# Generate token for different user
USER_ID=user-456 make token-static
export JWT_TOKEN_2="<new-token>"

# Try to access user-123's project
curl -H "Authorization: Bearer $JWT_TOKEN_2" \
  http://localhost:3000/api/v1/projects/{project_id}
```

Expected response (403 Forbidden):
```json
{
  "error": "Forbidden",
  "message": "forbidden: project does not belong to user"
}
```

---

### Testing Secret Provider Strategy (Multi-Tenant)

#### 1. Start the Server
```bash
make run-secret
```

Expected output shows configured tenants (tenant-1, tenant-2).

#### 2. Generate Tokens for Different Tenants
```bash
# Token for tenant-1, user alice
TENANT=tenant-1 USER_ID=alice make token-secret
export TOKEN_ALICE="<token>"

# Token for tenant-2, user bob
TENANT=tenant-2 USER_ID=bob make token-secret
export TOKEN_BOB="<token>"

# Another user in tenant-1
TENANT=tenant-1 USER_ID=charlie make token-secret
export TOKEN_CHARLIE="<token>"
```

#### 3. Create Projects for Each Tenant
```bash
# Alice (tenant-1) creates a project
curl -X POST http://localhost:3001/api/v1/projects \
  -H "Authorization: Bearer $TOKEN_ALICE" \
  -H "Content-Type: application/json" \
  -d '{"name": "Tenant 1 Project", "description": "Alice project"}'

# Bob (tenant-2) creates a project
curl -X POST http://localhost:3001/api/v1/projects \
  -H "Authorization: Bearer $TOKEN_BOB" \
  -H "Content-Type: application/json" \
  -d '{"name": "Tenant 2 Project", "description": "Bob project"}'

# Charlie (tenant-1) creates a project
curl -X POST http://localhost:3001/api/v1/projects \
  -H "Authorization: Bearer $TOKEN_CHARLIE" \
  -H "Content-Type: application/json" \
  -d '{"name": "Charlie Project", "description": "Another tenant-1 project"}'
```

#### 4. Verify Tenant Isolation
```bash
# Alice lists projects (should see only tenant-1:alice projects)
curl -H "Authorization: Bearer $TOKEN_ALICE" \
  http://localhost:3001/api/v1/projects

# Bob lists projects (should see only tenant-2:bob projects)
curl -H "Authorization: Bearer $TOKEN_BOB" \
  http://localhost:3001/api/v1/projects

# Charlie lists projects (should see only tenant-1:charlie projects)
curl -H "Authorization: Bearer $TOKEN_CHARLIE" \
  http://localhost:3001/api/v1/projects
```

Note: Each user sees only their own projects. Even though Alice and Charlie are in the same tenant, they have separate project lists because the user ID is composite: `issuer:sub`.

#### 5. Test Cross-Tenant Access (Should Fail)
```bash
# Get Alice's project ID
ALICE_PROJECT_ID="<from-create-response>"

# Try to access with Bob's token (different tenant)
curl -H "Authorization: Bearer $TOKEN_BOB" \
  http://localhost:3001/api/v1/projects/$ALICE_PROJECT_ID
```

Expected response (403 Forbidden): Bob cannot access Alice's project.

#### 6. Test Invalid Tenant Token
```bash
# Try to use a token with wrong issuer
curl -X POST http://localhost:3001/api/v1/projects \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Should Fail"}'
```

Expected response (401 Unauthorized): Unknown issuer.

---

### Testing JWKS Strategy (RS256)

#### 1. Start the Server
```bash
make run-jwks
```

Expected output shows:
- HTTP server on port 3002
- Mock JWKS server on port 9000
- JWKS URL: http://localhost:9000/.well-known/jwks.json

#### 2. Verify JWKS Endpoint
```bash
curl http://localhost:9000/.well-known/jwks.json
```

Expected response:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "mock-key-1",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

#### 3. Generate an RS256 Token
```bash
make token-jwks
export JWT_TOKEN="<token>"
```

#### 4. Create a Project with RS256 Token
```bash
curl -X POST http://localhost:3002/api/v1/projects \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "JWKS Project",
    "description": "Testing RS256 validation"
  }'
```

Expected response (201 Created): Project created successfully.

#### 5. List Projects
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3002/api/v1/projects
```

#### 6. Get Project
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3002/api/v1/projects/{project_id}
```

#### 7. Update Project
```bash
curl -X PUT http://localhost:3002/api/v1/projects/{project_id} \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated JWKS Project"
  }'
```

#### 8. Delete Project
```bash
curl -X DELETE http://localhost:3002/api/v1/projects/{project_id} \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### 9. Test with Different User
```bash
# Generate token for different user
USER_ID=user-789 make token-jwks
export JWT_TOKEN_2="<token>"

# Create project with new token
curl -X POST http://localhost:3002/api/v1/projects \
  -H "Authorization: Bearer $JWT_TOKEN_2" \
  -H "Content-Type: application/json" \
  -d '{"name": "User 789 Project"}'
```

#### 10. Verify User Isolation
```bash
# User 789 tries to access user-123's project
curl -H "Authorization: Bearer $JWT_TOKEN_2" \
  http://localhost:3002/api/v1/projects/{project_id}
```

Expected response (403 Forbidden): Users are isolated.

---

## Environment Variables

### Static Secret Strategy

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3000` | HTTP server port |
| `JWT_SECRET` | `dev-secret-change-in-production` | HMAC secret for signing/validating tokens |
| `JWT_ISSUER` | `jwt-example-static` | Expected issuer claim |

### Secret Provider Strategy

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3001` | HTTP server port |
| `TENANT_1_NAME` | `tenant-1` | First tenant identifier |
| `TENANT_1_SECRET` | `secret-tenant-1-change-in-production` | First tenant's HMAC secret |
| `TENANT_2_NAME` | `tenant-2` | Second tenant identifier |
| `TENANT_2_SECRET` | `secret-tenant-2-change-in-production` | Second tenant's HMAC secret |

Add more tenants with `TENANT_N_NAME` and `TENANT_N_SECRET` (N = 3, 4, 5, ...).

### JWKS Strategy

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3002` | HTTP server port |
| `JWKS_URL` | Auto (mock server) | JWKS endpoint URL |
| `JWT_ISSUER` | `mock-jwks-issuer` | Expected issuer claim |
| `JWT_AUDIENCE` | `mock-jwks-audience` | Expected audience claim |
| `USE_MOCK_JWKS` | `false` | Force use of mock JWKS server |

### Token Generator Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USER_ID` | `user-123` | User identifier (sub claim) |
| `USER_EMAIL` | `{USER_ID}@example.com` | User email |
| `TENANT` | `tenant-1` | Tenant name (secret provider only) |

## API Endpoints

### Health Check
```
GET /health
```
No authentication required.

### Projects API

All endpoints require JWT authentication via `Authorization: Bearer <token>` header.

#### Create Project
```
POST /api/v1/projects
Content-Type: application/json

{
  "name": "Project Name",
  "description": "Optional description"
}
```

Validation:
- `name` is required, 1-100 characters
- `description` is optional, max 500 characters
- `name` must be unique for the authenticated user

#### List Projects
```
GET /api/v1/projects
```

Returns all projects owned by the authenticated user.

#### Get Project
```
GET /api/v1/projects/{id}
```

Returns a specific project if owned by the authenticated user.

#### Update Project
```
PUT /api/v1/projects/{id}
Content-Type: application/json

{
  "name": "New Name",           // optional
  "description": "New Description"  // optional
}
```

At least one field must be provided.

#### Delete Project
```
DELETE /api/v1/projects/{id}
```

Deletes the project if owned by the authenticated user.

## Testing

### Unit Tests
```bash
make test-unit
```

Runs tests for:
- Repository layer (100% coverage)
- Service handlers (≥90% coverage)
- HTTP handlers (≥85% coverage)

### Integration Tests
```bash
make test-integration
```

Runs end-to-end tests for all three strategies:
- Static secret: 12 test cases
- Secret provider: 10 test cases (multi-tenant isolation)
- JWKS: 8 test cases (RS256 validation)

### Coverage Report
```bash
make test-coverage
```

Generates a coverage report and displays coverage percentages.

View HTML coverage report:
```bash
go tool cover -html=coverage.out
```

## When to Use Each Strategy

### Static Secret (HS256)
**Use When**:
- Building internal tools or admin panels
- Single-tenant applications
- Development and testing environments
- You control both token generation and validation

**Avoid When**:
- Building multi-tenant SaaS applications
- Integrating with external identity providers
- You need to distribute public keys

### Secret Provider (Multi-Tenant)
**Use When**:
- Building B2B SaaS platforms with tenant isolation
- Each tenant needs separate authentication
- You want HMAC simplicity with multi-tenant support
- You control token generation for all tenants

**Avoid When**:
- Integrating with OAuth 2.0/OIDC providers
- You need to share validation capabilities publicly
- Tenants generate their own tokens

### JWKS (RS256)
**Use When**:
- Integrating with Auth0, Okta, Keycloak, or similar
- Building OAuth 2.0/OIDC resource servers
- Microservices architecture with distributed validation
- You need to publish public keys for validation
- Zero-trust security model

**Avoid When**:
- You need the simplest possible solution
- Performance is critical (HMAC is faster than RSA)
- You control both sides of authentication

## Architecture Notes

This example demonstrates:

1. **Hexagonal Architecture**:
   - Domain layer: Pure business logic (`domain/project`)
   - Service layer: Application logic (`modules/project`)
   - Adapter layer: HTTP/transport (`modules/http`)

2. **Mono Framework Patterns**:
   - `ServiceProviderModule`: Project module exposes services
   - `DependentModule`: HTTP module depends on project services
   - `MiddlewareModule`: JWT middleware intercepts requests
   - Inter-module communication via NATS

3. **Thread Safety**:
   - Repository uses `sync.RWMutex` for concurrent access
   - Secret store uses `sync.RWMutex` for thread-safe lookups

4. **Composite User Identity**:
   - Format: `issuer:sub` (e.g., `tenant-1:alice`)
   - Ensures tenant isolation in multi-tenant scenarios
   - Allows same user ID across different tenants

## Troubleshooting

### Server Won't Start
- Check if port is already in use: `lsof -i :3000` (or 3001, 3002)
- Kill process: `kill -9 <pid>`

### Token Invalid
- Verify token hasn't expired (default: 1 hour)
- Check secret/issuer matches between generator and server
- For JWKS: Ensure mock server is running

### 403 Forbidden
- User ID in token doesn't match project owner
- For multi-tenant: Check issuer in token matches project owner's issuer

### Tests Failing
- Ensure no servers are running on ports 3000-3002, 9000-9001
- Run `make clean` and retry
- Check Go version: `go version` (requires 1.21+)

## Contributing

This is an example project for demonstration purposes. For production use:

1. Replace mock secrets with secure random values
2. Use environment-specific configuration
3. Add rate limiting and request validation
4. Implement proper logging and monitoring
5. Add API versioning strategy
6. Consider adding refresh tokens
7. Implement token revocation

## License

This example is part of the Mono Framework and follows the same license.
