# JWT Middleware Example - Technical Design

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP Layer (Fiber)                          │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  GET /health                                                   │ │
│  │  POST   /api/v1/projects        ┐                             │ │
│  │  GET    /api/v1/projects/:id    │                             │ │
│  │  GET    /api/v1/projects        ├─ Authorization Required     │ │
│  │  PUT    /api/v1/projects/:id    │                             │ │
│  │  DELETE /api/v1/projects/:id    ┘                             │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ HTTP Module (DependentModule)
                      │ Depends on: ["project"]
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Project Adapter (Port)                         │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  Create(ctx, CreateProjectRequest) → ProjectResponse          │ │
│  │  Get(ctx, GetProjectRequest) → ProjectResponse                │ │
│  │  List(ctx, ListProjectsRequest) → ListProjectsResponse        │ │
│  │  Update(ctx, UpdateProjectRequest) → ProjectResponse          │ │
│  │  Delete(ctx, DeleteProjectRequest) → ProjectResponse          │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ Calls via mono.ServiceContainer
                      │ Creates mono.Msg with Authorization header
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    JWT Middleware (Intercepts)                      │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  OnServiceRegistration() wraps handlers                       │ │
│  │  1. Extract JWT from msg.Header["Authorization"]              │ │
│  │  2. Validate signature (HS256/RS256 depending on strategy)    │ │
│  │  3. Validate claims (exp, nbf, iss, aud)                      │ │
│  │  4. Inject claims into context                                │ │
│  │  5. Call original handler with enhanced context               │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ Enhanced context with claims
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Project Module (ServiceProviderModule)            │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  RegisterServices(container):                                 │ │
│  │    - services.project.create                                  │ │
│  │    - services.project.get                                     │ │
│  │    - services.project.list                                    │ │
│  │    - services.project.update                                  │ │
│  │    - services.project.delete                                  │ │
│  └───────────────────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  Service Handlers:                                            │ │
│  │    - Extract user ID from context claims                      │ │
│  │    - Validate business rules                                  │ │
│  │    - Call repository for data operations                      │ │
│  │    - Check authorization (owner-based access)                 │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ Repository calls
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  Mock Repository (In-Memory)                        │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  map[string]*Project  (ID → Project)                          │ │
│  │  map[string][]string  (OwnerID → []ProjectID)                 │ │
│  │  sync.RWMutex         (Thread-safe)                           │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. Domain Layer

**Location**: `domain/project/`

#### Entity (`entity.go`)

```go
type Project struct {
    ID          string    `json:"id"`           // UUID
    Name        string    `json:"name"`         // 1-100 characters
    Description string    `json:"description"`  // Max 500 characters
    OwnerID     string    `json:"owner_id"`     // From JWT sub claim
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Business Rules**:
- ID: Auto-generated UUID
- Name: Required, unique per owner
- OwnerID: Set from JWT claims, immutable
- Timestamps: Auto-managed

#### Types (`types.go`)

Request/Response DTOs for service communication:

```go
type CreateProjectRequest struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

type UpdateProjectRequest struct {
    ID          string  `json:"id"`
    Name        *string `json:"name,omitempty"`        // Partial update
    Description *string `json:"description,omitempty"`  // Partial update
}

type ProjectResponse struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    OwnerID     string `json:"owner_id"`
    CreatedAt   string `json:"created_at"`  // RFC3339 format
    UpdatedAt   string `json:"updated_at"`  // RFC3339 format
}

type ListProjectsResponse struct {
    Projects []ProjectResponse `json:"projects"`
    Total    int               `json:"total"`
}
```

### 2. Project Module

**Location**: `modules/project/`

#### Module (`module.go`)

```go
type Module struct {
    repo *Repository
}

// Interfaces implemented:
// - mono.Module
// - mono.ServiceProviderModule

func (m *Module) Name() string { return "project" }

func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    // Register 5 typed request-reply services
}
```

**Service Registration**:
- Subject: `services.project.create`
- Subject: `services.project.get`
- Subject: `services.project.list`
- Subject: `services.project.update`
- Subject: `services.project.delete`

#### Service Handlers (`service.go`)

All handlers follow this pattern:

```go
func (m *Module) createProject(
    ctx context.Context,
    req domain.CreateProjectRequest,
    msg *mono.Msg,
) (domain.ProjectResponse, error) {
    // 1. Extract user ID from JWT claims in context
    userID, err := getUserIDFromContext(ctx)

    // 2. Validate request
    if req.Name == "" {
        return domain.ProjectResponse{}, fmt.Errorf("name is required")
    }

    // 3. Check business rules (e.g., duplicate name)
    exists, _ := m.repo.ExistsByOwnerAndName(userID, req.Name)
    if exists {
        return domain.ProjectResponse{}, fmt.Errorf("project already exists")
    }

    // 4. Create entity
    project := &domain.Project{
        ID:      uuid.New().String(),
        Name:    req.Name,
        OwnerID: userID,
        // ...
    }

    // 5. Save to repository
    m.repo.Create(project)

    // 6. Return response
    return toProjectResponse(project), nil
}
```

**Authorization Pattern**:
- Extract `userID` from context claims
- For get/update/delete: Verify `project.OwnerID == userID`
- For list: Filter by `OwnerID == userID`
- Return 403 error if ownership check fails

#### Repository (`repository.go`)

```go
type Repository struct {
    mu       sync.RWMutex
    projects map[string]*domain.Project  // ID → Project
    byOwner  map[string][]string         // OwnerID → []ProjectID
}

// Thread-safe operations:
// - Create(project)
// - FindByID(id)
// - FindByOwner(ownerID)
// - ExistsByOwnerAndName(ownerID, name)
// - Update(project)
// - Delete(id)
```

**Concurrency**:
- Use `RLock()` for reads
- Use `Lock()` for writes
- Maintain index: `byOwner` for efficient owner-based queries

#### Adapter (`adapter.go`)

```go
type ProjectPort interface {
    Create(ctx, CreateProjectRequest) (ProjectResponse, error)
    Get(ctx, GetProjectRequest) (ProjectResponse, error)
    List(ctx, ListProjectsRequest) (ListProjectsResponse, error)
    Update(ctx, UpdateProjectRequest) (ProjectResponse, error)
    Delete(ctx, DeleteProjectRequest) (ProjectResponse, error)
}

type Adapter struct {
    container mono.ServiceContainer
}

func (a *Adapter) Create(ctx context.Context, req domain.CreateProjectRequest) (domain.ProjectResponse, error) {
    client, _ := a.container.GetRequestReplyService("create")
    data, _ := json.Marshal(req)

    // Call service via NATS
    resp, err := client.Call(ctx, data)

    // Unmarshal response
    var result domain.ProjectResponse
    json.Unmarshal(resp.Data, &result)
    return result, nil
}
```

**Purpose**: Decouple HTTP module from project module internals

### 3. HTTP Module

**Location**: `modules/http/`

#### Module (`module.go`)

```go
type Module struct {
    app     *fiber.App
    port    int
    project project.ProjectPort  // Adapter interface
}

// Interfaces implemented:
// - mono.Module
// - mono.DependentModule
// - mono.HealthCheckableModule

func (m *Module) Dependencies() []string {
    return []string{"project"}
}

func (m *Module) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
    if dep == "project" {
        m.project = project.NewAdapter(container)
    }
}
```

**Lifecycle**:
- `Start()`: Create Fiber app, setup routes, start server in goroutine
- `Stop()`: Call `app.ShutdownWithContext(ctx)` for graceful shutdown

#### Handlers (`handlers.go`)

```go
func (m *Module) createProjectHandler(c *fiber.Ctx) error {
    // 1. Parse request body
    var req domain.CreateProjectRequest
    c.BodyParser(&req)

    // 2. Call project service via adapter
    // The adapter will create a mono.Msg with the Authorization header
    // from the HTTP request context
    resp, err := m.project.Create(c.Context(), req)

    // 3. Handle errors and map to HTTP status codes
    if err != nil {
        if isUnauthorizedError(err) {
            return c.Status(401).JSON(ErrorResponse{...})
        }
        // ... other error mappings
    }

    // 4. Return success response
    return c.Status(201).JSON(resp)
}
```

**JWT Propagation**:
The HTTP module must ensure the `Authorization` header from HTTP requests is available in the context when calling project services. The JWT middleware will then extract and validate it.

#### Middleware (`middleware.go`)

Error classification helpers:

```go
func isUnauthorizedError(err error) bool {
    msg := err.Error()
    return strings.Contains(msg, "no JWT claims") ||
           strings.Contains(msg, "invalid token")
}

func isForbiddenError(err error) bool {
    return strings.Contains(err.Error(), "forbidden")
}
```

### 4. JWT Middleware Integration

**Source**: `/home/leo/Projects/myspec/mono_framework/contrib/v1/jwt`

The middleware wraps all service handlers:

```go
// In each cmd/*/main.go
jwtMiddleware, _ := jwt.New(
    jwt.WithSecret([]byte("secret")),  // or WithJWKS(), etc.
)

app.Register(jwtMiddleware)  // Register BEFORE project module
app.Register(project.NewModule())
app.Register(http.NewModule(port))
```

**Execution Flow**:
1. HTTP request arrives with `Authorization: Bearer <token>`
2. HTTP handler calls `project.Create(ctx, req)`
3. Adapter creates `mono.Msg` with header from request
4. JWT middleware intercepts handler
5. Middleware extracts token, validates, injects claims into context
6. Original handler executes with enhanced context
7. Handler extracts user ID from claims
8. Response flows back through the chain

### 5. JWT Strategies

#### Static Secret (`cmd/static/main.go`)

```go
jwtMiddleware, _ := jwt.New(
    jwt.WithSecret([]byte(jwtSecret)),
    jwt.WithIssuer(jwtIssuer),
)
```

- **Algorithm**: HS256
- **Secret**: Single shared secret from env var
- **Use Case**: Development, testing, single-tenant apps

#### Secret Provider (`cmd/secret/main.go`)

```go
secretProvider := func(issuer string) ([]byte, error) {
    return secretStore.GetSecret(issuer)
}

jwtMiddleware, _ := jwt.New(
    jwt.WithSecretProvider(secretProvider),
)
```

- **Algorithm**: HS256
- **Secret**: Per-issuer lookup from map
- **Use Case**: Multi-tenant, per-customer secrets

#### JWKS (`cmd/jwks/main.go`)

```go
jwtMiddleware, _ := jwt.New(
    jwt.WithJWKS(jwksURL),
    jwt.WithIssuer(issuer),
    jwt.WithAudience(audience),
)
```

- **Algorithm**: RS256
- **Keys**: Fetched from JWKS endpoint (or mock server)
- **Use Case**: OAuth2 providers, public key cryptography

## Data Flow Diagrams

### Create Project Flow

```
1. HTTP Request
   POST /api/v1/projects
   Authorization: Bearer eyJhbGc...
   {"name": "My Project"}
   │
   ▼
2. HTTP Handler (createProjectHandler)
   Parse body → CreateProjectRequest
   │
   ▼
3. Project Adapter
   adapter.Create(ctx, req)
   │
   ▼
4. Request-Reply Service Call
   Subject: services.project.create
   Msg.Header["Authorization"] = ["Bearer eyJhbGc..."]
   Msg.Data = JSON(req)
   │
   ▼
5. JWT Middleware (intercepts)
   Extract token from Header
   Validate signature
   Parse claims: {"sub": "user-123", ...}
   Inject claims into context
   │
   ▼
6. Project Service Handler (createProject)
   Extract userID from context.claims
   Validate: name is required
   Check: project name unique for user
   Create entity with OwnerID = userID
   Save to repository
   │
   ▼
7. Repository
   Lock
   projects[uuid] = project
   byOwner[userID].append(uuid)
   Unlock
   │
   ▼
8. Response Flow
   ProjectResponse ← Service
   JSON ← Adapter
   HTTP 201 ← Handler
   │
   ▼
9. HTTP Response
   {"id": "...", "name": "My Project", "owner_id": "user-123"}
```

### Authorization Check Flow

```
User 1 requests User 2's project:

1. HTTP GET /api/v1/projects/proj-456
   Authorization: Bearer <user-1-token>
   │
   ▼
2. JWT Middleware
   Validate token
   Claims: {"sub": "user-1"}
   │
   ▼
3. Get Project Handler
   userID = "user-1" (from claims)
   project = repository.FindByID("proj-456")
   project.OwnerID = "user-2"
   │
   ▼
4. Authorization Check
   if project.OwnerID != userID {
       return error("forbidden: you do not own this project")
   }
   │
   ▼
5. HTTP Response
   403 Forbidden
   {"error": "Forbidden", "message": "forbidden: you do not own this project"}
```

## Module Registration Order

Critical for correct middleware wrapping:

```go
func main() {
    app, _ := mono.NewMonoApplication(...)

    // 1. JWT Middleware - MUST be first to wrap handlers
    app.Register(jwtMiddleware)

    // 2. Project Module - Provides services (will be wrapped)
    app.Register(project.NewModule())

    // 3. HTTP Module - Depends on project services
    app.Register(http.NewModule(port))

    app.Start(ctx)
}
```

**Why this order?**
- Middleware registers first to intercept `OnServiceRegistration` events
- Project module registers services (middleware wraps them)
- HTTP module depends on project services (already wrapped)

## Token Generation Design

Each strategy includes a standalone token generator:

```
cmd/
├── static/
│   ├── main.go                 # Main application
│   └── generate_token.go       # Token generator
├── secret/
│   ├── main.go
│   └── generate_token.go
└── jwks/
    ├── main.go
    ├── generate_token.go
    └── mock_jwks_server.go     # Mock JWKS server
```

**Token Generator Features**:
- Standalone execution: `go run cmd/<strategy>/generate_token.go`
- Environment variable configuration
- Human-readable output with usage instructions
- JSON output with `--json` flag
- Valid for 1 hour (configurable)

## Error Handling Design

### Service Layer

```go
// Clear, wrapped errors
return fmt.Errorf("failed to create project: %w", err)
return fmt.Errorf("project not found")
return fmt.Errorf("forbidden: you do not own this project")
```

### HTTP Layer

```go
// Map service errors to HTTP status codes
switch {
case isUnauthorizedError(err):
    return c.Status(401).JSON(ErrorResponse{
        Error:   "Unauthorized",
        Message: err.Error(),
    })
case isForbiddenError(err):
    return c.Status(403).JSON(...)
case isNotFoundError(err):
    return c.Status(404).JSON(...)
default:
    return c.Status(500).JSON(...)
}
```

## Testing Strategy

### Unit Tests

**Repository Tests**: 100% coverage
- CRUD operations
- Concurrent access (goroutine safety)
- Edge cases (not found, duplicates)

**Service Tests**: 90%+ coverage
- Valid requests
- Missing JWT claims
- Validation errors
- Authorization failures
- Business rule violations

**HTTP Handler Tests**: 85%+ coverage
- Valid requests
- Invalid JSON
- Missing authentication
- Error status code mapping

### Integration Tests

**Per-Strategy Tests**:
- Start real Mono application
- Register middleware + modules
- Generate valid token
- Test full HTTP → Service → Repository flow
- Verify authentication failures

## Configuration Design

All configuration via environment variables:

```bash
# Common
HTTP_PORT=3000
SHUTDOWN_TIMEOUT=30

# Static Secret
JWT_SECRET=dev-secret-change-in-production
JWT_ISSUER=jwt-example-static

# Secret Provider
TENANT_1_NAME=tenant-1
TENANT_1_SECRET=tenant-1-secret

# JWKS
JWKS_URL=https://example.auth0.com/.well-known/jwks.json
JWT_ISSUER=https://example.auth0.com/
JWT_AUDIENCE=my-api-identifier
USE_MOCK_JWKS=true  # For demo
```

**No configuration files** - keep it simple for examples

## Build System Design

Makefile provides:

```makefile
# Build
build, build-static, build-secret, build-jwks

# Run
run-static, run-secret, run-jwks, run-jwks-real

# Token Generation
token-static, token-secret, tokens-secret, token-jwks

# Testing
test, test-unit, test-integration, test-coverage

# Code Quality
fmt, lint, tidy, check

# Cleanup
clean
```

Color-coded output for better UX.

---

*Last Updated: January 2026*
*Version: 1.0*
