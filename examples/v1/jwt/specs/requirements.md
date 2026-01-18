# JWT Middleware Example - Requirements

## Functional Requirements

### FR1: Domain Model

**FR1.1: Project Entity**
- The system SHALL provide a `Project` entity with the following fields:
  - `ID` (string, UUID format)
  - `Name` (string, 1-100 characters)
  - `Description` (string, max 500 characters)
  - `OwnerID` (string, from JWT sub claim)
  - `CreatedAt` (timestamp)
  - `UpdatedAt` (timestamp)

**FR1.2: Business Rules**
- Project names SHALL be unique per owner
- Project names MUST NOT exceed 100 characters
- Descriptions MUST NOT exceed 500 characters
- OwnerID SHALL be immutable after creation
- Timestamps SHALL be automatically managed

### FR2: CRUD Operations

**FR2.1: Create Project**
- Users SHALL be able to create new projects
- Input: Name (required), Description (optional)
- The system SHALL auto-generate a unique UUID for the project ID
- The system SHALL set OwnerID from the JWT sub claim
- The system SHALL return HTTP 400 if name is empty
- The system SHALL return HTTP 400 if a project with the same name already exists for the user
- The system SHALL return HTTP 201 with the created project on success

**FR2.2: Get Project by ID**
- Users SHALL be able to retrieve a project by its ID
- Input: Project ID
- The system SHALL return HTTP 404 if the project does not exist
- The system SHALL return HTTP 403 if the user does not own the project
- The system SHALL return HTTP 200 with the project data on success

**FR2.3: List Projects**
- Users SHALL be able to list all their projects
- The system SHALL return only projects owned by the authenticated user
- The system SHALL return an empty array if the user has no projects
- The system SHALL return HTTP 200 with a list of projects and total count

**FR2.4: Update Project**
- Users SHALL be able to update their own projects
- Input: Project ID, Name (optional), Description (optional)
- The system SHALL support partial updates (only provided fields are updated)
- The system SHALL update the `UpdatedAt` timestamp
- The system SHALL return HTTP 404 if the project does not exist
- The system SHALL return HTTP 403 if the user does not own the project
- The system SHALL return HTTP 400 if the new name already exists for the user
- The system SHALL return HTTP 400 if no fields are provided for update
- The system SHALL return HTTP 200 with the updated project on success

**FR2.5: Delete Project**
- Users SHALL be able to delete their own projects
- Input: Project ID
- The system SHALL return HTTP 404 if the project does not exist
- The system SHALL return HTTP 403 if the user does not own the project
- The system SHALL return HTTP 200 with the deleted project data on success

### FR3: JWT Authentication

**FR3.1: Static Secret Strategy**
- The system SHALL support JWT authentication using a single shared HMAC secret
- The system SHALL use HS256 algorithm for signature validation
- The system SHALL read the secret from the `JWT_SECRET` environment variable
- The system SHALL validate the issuer claim if configured
- The system SHALL provide a token generation helper for testing

**FR3.2: Secret Provider Strategy**
- The system SHALL support JWT authentication with per-tenant secrets
- The system SHALL use HS256 algorithm with dynamic secret lookup
- The system SHALL look up the secret based on the token's `iss` claim
- The system SHALL support at least 3 different tenants
- The system SHALL return HTTP 401 if the issuer is unknown
- The system SHALL provide token generation helpers for each tenant

**FR3.3: JWKS Strategy**
- The system SHALL support JWT authentication using JWKS (JSON Web Key Set)
- The system SHALL use RS256 algorithm for signature validation
- The system SHALL fetch public keys from a JWKS endpoint
- The system SHALL validate `iss`, `aud`, and `kid` claims
- The system SHALL provide a mock JWKS server for demonstration
- The system SHALL provide a token generation helper using RSA key pair

**FR3.4: Token Validation**
- The system SHALL validate the JWT signature before handler execution
- The system SHALL validate the token expiration (`exp` claim)
- The system SHALL validate the not-before time (`nbf` claim) if present
- The system SHALL extract the subject (`sub` claim) as the user ID
- The system SHALL inject validated claims into the request context
- The system SHALL return HTTP 401 for missing Authorization header
- The system SHALL return HTTP 401 for invalid token format
- The system SHALL return HTTP 401 for expired tokens
- The system SHALL return HTTP 401 for invalid signatures

### FR4: Authorization

**FR4.1: Owner-Based Access Control**
- The system SHALL allow users to access only their own projects
- The system SHALL extract the user ID from the JWT sub claim
- The system SHALL compare the project's OwnerID with the authenticated user ID
- The system SHALL return HTTP 403 if the user attempts to access another user's project

### FR5: REST API

**FR5.1: HTTP Endpoints**
- The system SHALL provide the following endpoints:
  - `GET /health` - Health check (no authentication required)
  - `POST /api/v1/projects` - Create project (authentication required)
  - `GET /api/v1/projects/:id` - Get project (authentication required)
  - `GET /api/v1/projects` - List projects (authentication required)
  - `PUT /api/v1/projects/:id` - Update project (authentication required)
  - `DELETE /api/v1/projects/:id` - Delete project (authentication required)

**FR5.2: Request/Response Format**
- The system SHALL accept and return JSON format only
- The system SHALL use `Content-Type: application/json` for all requests
- The system SHALL return errors in the format: `{"error": "...", "message": "..."}`
- The system SHALL return project data in `ProjectResponse` format
- The system SHALL return timestamps in RFC3339 format

**FR5.3: HTTP Status Codes**
- The system SHALL return appropriate HTTP status codes:
  - `200 OK` - Successful GET, PUT, DELETE
  - `201 Created` - Successful POST
  - `400 Bad Request` - Validation errors
  - `401 Unauthorized` - Missing or invalid JWT
  - `403 Forbidden` - Authorization failures
  - `404 Not Found` - Resource not found
  - `500 Internal Server Error` - Unexpected errors

### FR6: Token Generation

**FR6.1: Token Generator Tools**
- The system SHALL provide token generation tools for each JWT strategy
- Token generators SHALL be executable with `go run cmd/<strategy>/generate_token.go`
- Token generators SHALL support environment variable configuration
- Token generators SHALL output valid JWT tokens for testing
- Token generators SHALL include usage instructions

**FR6.2: Token Configuration**
- Static secret generator SHALL support `USER_ID` and `USER_EMAIL` env vars
- Secret provider generator SHALL support `TENANT`, `USER_ID`, and `USER_EMAIL` env vars
- JWKS generator SHALL use the mock server's key pair
- All generators SHALL support `--json` flag for machine-readable output
- All tokens SHALL be valid for 1 hour by default

### FR7: Data Persistence

**FR7.1: Mock Repository**
- The system SHALL use an in-memory repository (no external database)
- The repository SHALL store projects in a thread-safe map
- The repository SHALL maintain an index of projects by owner ID
- The repository SHALL support concurrent read/write operations
- The repository SHALL lose all data on application restart

### FR8: Module System

**FR8.1: Project Module**
- The system SHALL implement a project module as a `ServiceProviderModule`
- The module SHALL register 5 request-reply services (create, get, list, update, delete)
- The module SHALL use typed service handlers with `helper.RegisterTypedRequestReplyService`
- The module SHALL validate all requests before processing
- The module SHALL enforce business rules (unique names, ownership)

**FR8.2: HTTP Module**
- The system SHALL implement an HTTP module as a `DependentModule`
- The module SHALL depend on the project module
- The module SHALL use the project adapter for service calls
- The module SHALL start a Fiber HTTP server
- The module SHALL support graceful shutdown

**FR8.3: Module Registration**
- The system SHALL register modules in the correct order:
  1. JWT middleware
  2. Project module
  3. HTTP module

## Non-Functional Requirements

### NFR1: Performance

**NFR1.1: Startup Time**
- The application SHALL start in less than 2 seconds

**NFR1.2: Request Latency**
- REST API requests SHALL complete in less than 100ms (excluding JWT validation)
- JWT validation SHALL complete in less than 10ms for cached keys

**NFR1.3: Concurrent Operations**
- The repository SHALL support concurrent read and write operations
- The system SHALL handle at least 100 concurrent requests without errors

### NFR2: Security

**NFR2.1: Token Security**
- The system SHALL NOT log full JWT tokens
- The system SHALL mask secrets in startup logs
- The system SHALL display security warnings for default secrets
- The system SHALL validate all JWT signatures before handler execution

**NFR2.2: Authorization Security**
- The system SHALL enforce owner-based access control for all operations
- The system SHALL NOT allow users to access other users' data
- The system SHALL return generic error messages that don't leak sensitive info

### NFR3: Reliability

**NFR3.1: Error Handling**
- The system SHALL handle all errors gracefully without crashing
- The system SHALL NOT use panic() in production code
- The system SHALL provide meaningful error messages
- The system SHALL log all errors with context

**NFR3.2: Graceful Shutdown**
- The system SHALL handle SIGINT and SIGTERM signals
- The system SHALL wait for in-flight requests to complete (up to 30 seconds)
- The system SHALL close all resources properly during shutdown

### NFR4: Maintainability

**NFR4.1: Code Quality**
- All code SHALL be formatted with `gofmt`
- All exported types and functions SHALL have godoc comments
- All code SHALL follow Go best practices and idioms
- All complex logic SHALL have explanatory comments

**NFR4.2: Test Coverage**
- Repository tests SHALL achieve 100% code coverage
- Service tests SHALL achieve >90% code coverage
- HTTP handler tests SHALL achieve >85% code coverage
- All tests SHALL pass before code is committed

**NFR4.3: Documentation**
- The system SHALL include a comprehensive README with usage instructions
- Each JWT strategy SHALL have clear documentation on when to use it
- Token generators SHALL output usage examples
- All API endpoints SHALL be documented with examples

### NFR5: Usability

**NFR5.1: Developer Experience**
- The Makefile SHALL provide intuitive commands for common tasks
- The Makefile SHALL use color output for better readability
- The Makefile SHALL include a help command that lists all targets
- Token generators SHALL output human-readable instructions

**NFR5.2: Configuration**
- All configuration SHALL be via environment variables
- All configuration SHALL have sensible defaults
- Environment variables SHALL be clearly documented

**NFR5.3: Error Messages**
- Error messages SHALL be clear and actionable
- HTTP error responses SHALL include both error code and message
- Validation errors SHALL specify which field is invalid

### NFR6: Portability

**NFR6.1: Platform Support**
- The system SHALL run on Linux and macOS
- The Makefile SHALL work on Linux and macOS
- Windows support SHALL be documented (e.g., "use WSL or Git Bash")

**NFR6.2: Dependencies**
- The system SHALL minimize external dependencies
- All dependencies SHALL be managed via `go.mod`
- The system SHALL NOT require external databases or services (except for JWKS demo)

### NFR7: Testability

**NFR7.1: Unit Testing**
- All components SHALL be unit testable
- All dependencies SHALL be mockable via interfaces
- Tests SHALL run in less than 5 seconds total

**NFR7.2: Integration Testing**
- Each JWT strategy SHALL have integration tests
- Integration tests SHALL use real Mono application instances
- Integration tests SHALL test full request flow (HTTP → Service → Repository)

### NFR8: Observability

**NFR8.1: Logging**
- The system SHALL log all authentication failures
- The system SHALL log all authorization failures
- The system SHALL log startup and shutdown events
- Logs SHALL include timestamps and context

**NFR8.2: Health Checks**
- The system SHALL provide a `/health` endpoint
- The health endpoint SHALL return 200 when the system is healthy
- The health endpoint SHALL NOT require authentication

### NFR9: Scalability

**NFR9.1: Concurrent Users**
- The system SHALL support multiple concurrent users
- The system SHALL maintain data isolation between users
- The system SHALL use thread-safe data structures

**NFR9.2: Data Volume**
- The system SHALL handle at least 1000 projects per user
- The system SHALL handle at least 100 users simultaneously

## Acceptance Criteria

### AC1: Build and Run
- ✅ All three binaries build successfully with `make build`
- ✅ Each binary starts without errors
- ✅ Each binary responds to health check within 1 second

### AC2: Authentication
- ✅ Valid tokens are accepted and users can create projects
- ✅ Invalid tokens are rejected with HTTP 401
- ✅ Missing tokens are rejected with HTTP 401
- ✅ Expired tokens are rejected with HTTP 401
- ✅ All three JWT strategies work correctly

### AC3: Authorization
- ✅ Users can create, read, update, delete their own projects
- ✅ Users cannot read other users' projects (HTTP 403)
- ✅ Users cannot update other users' projects (HTTP 403)
- ✅ Users cannot delete other users' projects (HTTP 403)

### AC4: Business Logic
- ✅ Project names are validated (required, max 100 chars)
- ✅ Descriptions are validated (max 500 chars)
- ✅ Duplicate project names for same user are rejected (HTTP 400)
- ✅ Different users can have projects with the same name

### AC5: Data Integrity
- ✅ Created projects have valid UUIDs
- ✅ OwnerID is set correctly from JWT claims
- ✅ Timestamps are set correctly
- ✅ Updates change UpdatedAt timestamp
- ✅ Deleted projects are removed from repository

### AC6: Error Handling
- ✅ Invalid JSON returns HTTP 400
- ✅ Missing required fields return HTTP 400
- ✅ Non-existent project IDs return HTTP 404
- ✅ Unexpected errors return HTTP 500
- ✅ Error messages are clear and helpful

### AC7: Testing
- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Coverage meets requirements (100%/90%/85%)
- ✅ Tests run in less than 5 seconds

### AC8: Documentation
- ✅ README includes quick start for each strategy
- ✅ README includes example curl commands
- ✅ Token generators output usage instructions
- ✅ Makefile help command works

### AC9: Code Quality
- ✅ All code is formatted with gofmt
- ✅ No linter errors (if golangci-lint is installed)
- ✅ All exported functions have godoc comments
- ✅ go mod tidy runs without changes

### AC10: Graceful Shutdown
- ✅ SIGINT/SIGTERM shut down the server gracefully
- ✅ In-flight requests complete before shutdown
- ✅ Resources are cleaned up properly

## Out of Scope

The following are explicitly OUT OF SCOPE for this example:

- ❌ User registration and login flows
- ❌ Password hashing and authentication
- ❌ Real database integration (PostgreSQL, MySQL, etc.)
- ❌ Token refresh mechanisms
- ❌ Password reset flows
- ❌ Email verification
- ❌ Rate limiting
- ❌ API versioning beyond v1
- ❌ Pagination for list endpoints
- ❌ Filtering and sorting
- ❌ Project sharing or collaboration
- ❌ Role-based access control (RBAC)
- ❌ WebSocket support
- ❌ GraphQL support
- ❌ Metrics and monitoring (Prometheus, etc.)
- ❌ Distributed tracing
- ❌ Docker containerization
- ❌ Kubernetes deployment
- ❌ CI/CD pipelines (beyond basic GitHub Actions example)

---

*Last Updated: January 2026*
*Version: 1.0*
