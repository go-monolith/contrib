# JWT Middleware Example - Implementation Tasks

## Overview

This document defines step-by-step implementation tasks organized into milestones. Each milestone builds on the previous one and includes tests to verify correct implementation.

**Implementation Strategy**:
1. Build foundation layer first (domain, repository)
2. Add business logic (services, modules)
3. Add HTTP layer
4. Add JWT strategies
5. Add tests at each step
6. Add tooling (Makefile, token generators)

---

## Milestone 1: Project Foundation

**Goal**: Set up Go module, domain models, and mock repository with tests.

### Task 1.1: Initialize Go Module

**Description**: Create `go.mod` and set up project structure.

**Steps**:
```bash
cd /home/leo/Projects/myspec/mono_framework/contrib/examples/v1/jwt
go mod init github.com/go-monolith/mono/contrib/examples/v1/jwt
mkdir -p domain/project modules/project modules/http cmd/static cmd/secret cmd/jwks
```

**Dependencies**:
```bash
go get github.com/go-monolith/mono@latest
go get github.com/gofiber/fiber/v2@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/google/uuid@latest
go get github.com/gelmium/graceful-shutdown@latest
go get github.com/stretchr/testify@latest
```

**Verification**:
- [ ] `go.mod` exists with correct module path
- [ ] All dependencies download successfully
- [ ] `go mod tidy` runs without errors

---

### Task 1.2: Create Domain Models

**Description**: Implement `Project` entity and request/response types.

**Files to Create**:
1. `domain/project/entity.go`
2. `domain/project/types.go`

**entity.go**:
```go
package project

import "time"

// Project represents a project entity
type Project struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    OwnerID     string    `json:"owner_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**types.go**:
```go
package project

// CreateProjectRequest is the request to create a new project
type CreateProjectRequest struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

// GetProjectRequest is the request to get a project by ID
type GetProjectRequest struct {
    ID string `json:"id"`
}

// ListProjectsRequest is the request to list projects
type ListProjectsRequest struct {
    // Empty for now
}

// UpdateProjectRequest is the request to update a project
type UpdateProjectRequest struct {
    ID          string  `json:"id"`
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
}

// DeleteProjectRequest is the request to delete a project
type DeleteProjectRequest struct {
    ID string `json:"id"`
}

// ProjectResponse is the response for a single project
type ProjectResponse struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    OwnerID     string `json:"owner_id"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}

// ListProjectsResponse is the response for listing projects
type ListProjectsResponse struct {
    Projects []ProjectResponse `json:"projects"`
    Total    int               `json:"total"`
}
```

**Verification**:
- [ ] Files compile without errors
- [ ] All types have proper JSON tags
- [ ] godoc comments exist for all exported types

---

### Task 1.3: Implement Mock Repository

**Description**: Create thread-safe in-memory repository for projects.

**Files to Create**:
1. `modules/project/repository.go`
2. `modules/project/repository_test.go`

**repository.go** (key methods):
```go
package project

import (
    "fmt"
    "sync"
    domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

type Repository struct {
    mu       sync.RWMutex
    projects map[string]*domain.Project
    byOwner  map[string][]string
}

func NewRepository() *Repository {
    return &Repository{
        projects: make(map[string]*domain.Project),
        byOwner:  make(map[string][]string),
    }
}

// Implement:
// - Create(project *domain.Project) error
// - FindByID(id string) (*domain.Project, error)
// - FindByOwner(ownerID string) ([]*domain.Project, error)
// - ExistsByOwnerAndName(ownerID, name string) (bool, error)
// - Update(project *domain.Project) error
// - Delete(id string) error
```

**Tests to Write** (`repository_test.go`):
```go
func TestRepository_Create(t *testing.T)
func TestRepository_Create_Duplicate(t *testing.T)
func TestRepository_FindByID(t *testing.T)
func TestRepository_FindByID_NotFound(t *testing.T)
func TestRepository_FindByOwner(t *testing.T)
func TestRepository_FindByOwner_Empty(t *testing.T)
func TestRepository_ExistsByOwnerAndName(t *testing.T)
func TestRepository_Update(t *testing.T)
func TestRepository_Update_NotFound(t *testing.T)
func TestRepository_Delete(t *testing.T)
func TestRepository_Delete_NotFound(t *testing.T)
func TestRepository_ConcurrentAccess(t *testing.T)
```

**Verification**:
- [ ] All repository methods implemented
- [ ] All tests pass: `go test ./modules/project -v`
- [ ] Code coverage ≥ 100%: `go test ./modules/project -cover`
- [ ] Concurrent test verifies thread safety

---

## Milestone 2: Project Module Services

**Goal**: Implement project module as `ServiceProviderModule` with CRUD services.

### Task 2.1: Create Service Helpers

**Description**: Implement helper functions for service handlers.

**Files to Create**:
1. `modules/project/service.go`

**service.go** (helpers):
```go
package project

import (
    "context"
    "fmt"
    "time"
    "github.com/go-monolith/mono/contrib/v1/jwt"
    domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

// getUserIDFromContext extracts user ID from JWT claims
func getUserIDFromContext(ctx context.Context) (string, error) {
    claims := jwt.GetClaimsFromContext(ctx)
    if claims == nil {
        return "", fmt.Errorf("no JWT claims in context")
    }

    sub, ok := claims["sub"]
    if !ok {
        return "", fmt.Errorf("missing 'sub' claim in JWT")
    }

    userID, ok := sub.(string)
    if !ok {
        return "", fmt.Errorf("'sub' claim is not a string")
    }

    return userID, nil
}

// toProjectResponse converts entity to response
func toProjectResponse(p *domain.Project) domain.ProjectResponse {
    return domain.ProjectResponse{
        ID:          p.ID,
        Name:        p.Name,
        Description: p.Description,
        OwnerID:     p.OwnerID,
        CreatedAt:   p.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
    }
}
```

**Verification**:
- [ ] Helpers compile without errors
- [ ] Functions have godoc comments

---

### Task 2.2: Implement Service Handlers

**Description**: Implement CRUD service handlers.

**Files to Update**:
1. `modules/project/service.go`

**Handlers to Implement**:
```go
// createProject creates a new project
func (m *Module) createProject(
    ctx context.Context,
    req domain.CreateProjectRequest,
    msg *mono.Msg,
) (domain.ProjectResponse, error)

// getProject retrieves a project by ID
func (m *Module) getProject(
    ctx context.Context,
    req domain.GetProjectRequest,
    msg *mono.Msg,
) (domain.ProjectResponse, error)

// listProjects lists all projects for the authenticated user
func (m *Module) listProjects(
    ctx context.Context,
    req domain.ListProjectsRequest,
    msg *mono.Msg,
) (domain.ListProjectsResponse, error)

// updateProject updates an existing project
func (m *Module) updateProject(
    ctx context.Context,
    req domain.UpdateProjectRequest,
    msg *mono.Msg,
) (domain.ProjectResponse, error)

// deleteProject deletes a project by ID
func (m *Module) deleteProject(
    ctx context.Context,
    req domain.DeleteProjectRequest,
    msg *mono.Msg,
) (domain.ProjectResponse, error)
```

**Implementation Requirements**:
- Extract user ID from context claims
- Validate all inputs
- Check business rules (unique names, ownership)
- Return descriptive errors
- Use repository for data operations

**Verification**:
- [ ] All handlers compile without errors
- [ ] Handlers follow consistent error handling patterns

---

### Task 2.3: Implement Project Module

**Description**: Create project module with service registration.

**Files to Create**:
1. `modules/project/module.go`

**module.go**:
```go
package project

import (
    "context"
    "encoding/json"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
)

type Module struct {
    repo *Repository
}

var _ mono.Module = (*Module)(nil)
var _ mono.ServiceProviderModule = (*Module)(nil)

func NewModule() *Module {
    return &Module{
        repo: NewRepository(),
    }
}

func (m *Module) Name() string {
    return "project"
}

func (m *Module) Start(ctx context.Context) error {
    return nil
}

func (m *Module) Stop(ctx context.Context) error {
    return nil
}

func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    // Register create service
    if err := helper.RegisterTypedRequestReplyService(
        container, "create", json.Unmarshal, json.Marshal, m.createProject,
    ); err != nil {
        return err
    }

    // Register get, list, update, delete services...

    return nil
}
```

**Verification**:
- [ ] Module compiles without errors
- [ ] Module implements required interfaces
- [ ] All 5 services are registered

---

### Task 2.4: Write Service Handler Tests

**Description**: Test service handlers with mocked context.

**Files to Create**:
1. `modules/project/service_test.go`

**Test Helper**:
```go
// mockContext creates a context with JWT claims
func mockContext(userID string) context.Context {
    claims := map[string]interface{}{
        "sub":   userID,
        "email": userID + "@example.com",
    }
    return jwt.SetClaimsInContext(context.Background(), claims)
}
```

**Tests to Write**:
```go
// Create
func TestCreateProject_Success(t *testing.T)
func TestCreateProject_MissingClaims(t *testing.T)
func TestCreateProject_EmptyName(t *testing.T)
func TestCreateProject_NameTooLong(t *testing.T)
func TestCreateProject_DescriptionTooLong(t *testing.T)
func TestCreateProject_DuplicateName(t *testing.T)

// Get
func TestGetProject_Success(t *testing.T)
func TestGetProject_NotFound(t *testing.T)
func TestGetProject_Forbidden(t *testing.T)

// List
func TestListProjects_Success(t *testing.T)
func TestListProjects_Empty(t *testing.T)

// Update
func TestUpdateProject_Success(t *testing.T)
func TestUpdateProject_PartialUpdate(t *testing.T)
func TestUpdateProject_NotFound(t *testing.T)
func TestUpdateProject_Forbidden(t *testing.T)
func TestUpdateProject_DuplicateName(t *testing.T)
func TestUpdateProject_NoFields(t *testing.T)

// Delete
func TestDeleteProject_Success(t *testing.T)
func TestDeleteProject_NotFound(t *testing.T)
func TestDeleteProject_Forbidden(t *testing.T)
```

**Verification**:
- [ ] All tests pass: `go test ./modules/project -v`
- [ ] Code coverage ≥ 90%: `go test ./modules/project -cover`
- [ ] Tests verify authorization (forbidden cases)
- [ ] Tests verify validation errors

---

### Task 2.5: Create Project Adapter

**Description**: Implement port adapter for HTTP module.

**Files to Create**:
1. `modules/project/adapter.go`

**adapter.go**:
```go
package project

import (
    "context"
    "encoding/json"
    "github.com/go-monolith/mono"
    domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

// ProjectPort defines the interface for project operations
type ProjectPort interface {
    Create(ctx context.Context, req domain.CreateProjectRequest) (domain.ProjectResponse, error)
    Get(ctx context.Context, req domain.GetProjectRequest) (domain.ProjectResponse, error)
    List(ctx context.Context, req domain.ListProjectsRequest) (domain.ListProjectsResponse, error)
    Update(ctx context.Context, req domain.UpdateProjectRequest) (domain.ProjectResponse, error)
    Delete(ctx context.Context, req domain.DeleteProjectRequest) (domain.ProjectResponse, error)
}

// Adapter implements ProjectPort by calling project services
type Adapter struct {
    container mono.ServiceContainer
}

func NewAdapter(container mono.ServiceContainer) *Adapter {
    return &Adapter{container: container}
}

// Implement all 5 methods
```

**Verification**:
- [ ] Adapter implements ProjectPort interface
- [ ] All methods compile without errors

---

## Milestone 3: HTTP Module

**Goal**: Implement HTTP module with Fiber REST API.

### Task 3.1: Implement HTTP Module Structure

**Description**: Create HTTP module with lifecycle methods.

**Files to Create**:
1. `modules/http/module.go`

**module.go**:
```go
package http

import (
    "context"
    "fmt"
    "log"
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
)

type Module struct {
    app     *fiber.App
    port    int
    project project.ProjectPort
}

var _ mono.Module = (*Module)(nil)
var _ mono.DependentModule = (*Module)(nil)
var _ mono.HealthCheckableModule = (*Module)(nil)

func NewModule(port int) *Module {
    return &Module{port: port}
}

func (m *Module) Name() string {
    return "http"
}

func (m *Module) Dependencies() []string {
    return []string{"project"}
}

func (m *Module) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
    if dep == "project" {
        m.project = project.NewAdapter(container)
    }
}

func (m *Module) Start(ctx context.Context) error {
    // Create Fiber app
    // Setup routes
    // Start server in goroutine
    return nil
}

func (m *Module) Stop(ctx context.Context) error {
    if m.app != nil {
        return m.app.ShutdownWithContext(ctx)
    }
    return nil
}

func (m *Module) Health(ctx context.Context) mono.HealthStatus {
    return mono.HealthStatus{
        Healthy: m.app != nil,
        Message: "operational",
        Details: map[string]any{"port": m.port},
    }
}
```

**Verification**:
- [ ] Module compiles without errors
- [ ] Module implements all required interfaces

---

### Task 3.2: Implement HTTP Handlers

**Description**: Create REST API handlers for projects.

**Files to Create**:
1. `modules/http/handlers.go`
2. `modules/http/middleware.go`

**handlers.go**:
```go
package http

import (
    "github.com/gofiber/fiber/v2"
    domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}

func (m *Module) healthHandler(c *fiber.Ctx) error {
    // Return health status
}

func (m *Module) createProjectHandler(c *fiber.Ctx) error {
    // Parse request, call service, return response
}

func (m *Module) getProjectHandler(c *fiber.Ctx) error {
    // Extract ID, call service, return response
}

func (m *Module) listProjectsHandler(c *fiber.Ctx) error {
    // Call service, return list
}

func (m *Module) updateProjectHandler(c *fiber.Ctx) error {
    // Extract ID, parse body, call service
}

func (m *Module) deleteProjectHandler(c *fiber.Ctx) error {
    // Extract ID, call service
}
```

**middleware.go**:
```go
package http

import "strings"

func isUnauthorizedError(err error) bool {
    msg := err.Error()
    return strings.Contains(msg, "no JWT claims") ||
           strings.Contains(msg, "missing 'sub' claim")
}

func isForbiddenError(err error) bool {
    return strings.Contains(err.Error(), "forbidden")
}

func isNotFoundError(err error) bool {
    return strings.Contains(err.Error(), "not found")
}
```

**Verification**:
- [ ] All handlers compile without errors
- [ ] Error classification helpers work correctly

---

### Task 3.3: Setup Routes and Middleware

**Description**: Configure Fiber routes and middleware.

**Files to Update**:
1. `modules/http/module.go`

**setupRoutes method**:
```go
func (m *Module) setupRoutes() {
    // Health check
    m.app.Get("/health", m.healthHandler)

    // API v1
    v1 := m.app.Group("/api/v1")

    // Project routes
    projects := v1.Group("/projects")
    projects.Post("", m.createProjectHandler)
    projects.Get("/:id", m.getProjectHandler)
    projects.Get("", m.listProjectsHandler)
    projects.Put("/:id", m.updateProjectHandler)
    projects.Delete("/:id", m.deleteProjectHandler)
}
```

**Start method**:
```go
func (m *Module) Start(ctx context.Context) error {
    m.app = fiber.New(fiber.Config{
        DisableStartupMessage: true,
        ErrorHandler: customErrorHandler,
    })

    // Add middleware
    m.app.Use(recover.New())
    m.app.Use(logger.New())
    m.app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowHeaders: "Origin, Content-Type, Accept, Authorization",
    }))

    m.setupRoutes()

    // Start server
    go func() {
        addr := fmt.Sprintf(":%d", m.port)
        if err := m.app.Listen(addr); err != nil {
            log.Printf("[http] Error: %v", err)
        }
    }()

    return nil
}
```

**Verification**:
- [ ] Routes are registered correctly
- [ ] Middleware is applied
- [ ] Server starts without errors

---

### Task 3.4: Write HTTP Handler Tests

**Description**: Test HTTP handlers with mocks.

**Files to Create**:
1. `modules/http/handlers_test.go`

**Tests to Write**:
```go
func TestHealthHandler(t *testing.T)
func TestCreateProjectHandler_Success(t *testing.T)
func TestCreateProjectHandler_InvalidJSON(t *testing.T)
func TestGetProjectHandler_Success(t *testing.T)
func TestGetProjectHandler_NotFound(t *testing.T)
func TestListProjectsHandler_Success(t *testing.T)
func TestUpdateProjectHandler_Success(t *testing.T)
func TestDeleteProjectHandler_Success(t *testing.T)
```

**Verification**:
- [ ] All tests pass: `go test ./modules/http -v`
- [ ] Code coverage ≥ 85%: `go test ./modules/http -cover`
- [ ] Tests use mock project adapter
- [ ] Tests verify HTTP status codes

---

## Milestone 4: Static Secret Strategy

**Goal**: Implement first JWT strategy example with static HMAC secret.

### Task 4.1: Implement Static Secret Main

**Description**: Create main application for static secret strategy.

**Files to Create**:
1. `cmd/static/main.go`

**main.go**:
```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/contrib/v1/jwt"
    "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
    "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
    gfshutdown "github.com/gelmium/graceful-shutdown"
)

const (
    defaultHTTPPort       = 3000
    defaultJWTSecret      = "dev-secret-change-in-production"
    defaultJWTIssuer      = "jwt-example-static"
    defaultShutdownTimeout = 30 * time.Second
)

func main() {
    // Load config from env
    httpPort := getEnvInt("HTTP_PORT", defaultHTTPPort)
    jwtSecret := getEnv("JWT_SECRET", defaultJWTSecret)
    jwtIssuer := getEnv("JWT_ISSUER", defaultJWTIssuer)

    // Create application
    app, _ := mono.NewMonoApplication(
        mono.WithShutdownTimeout(defaultShutdownTimeout),
        mono.WithLogLevel(mono.LogLevelInfo),
    )

    // Create JWT middleware
    jwtMiddleware, _ := jwt.New(
        jwt.WithSecret([]byte(jwtSecret)),
        jwt.WithIssuer(jwtIssuer),
    )

    // Register modules
    app.Register(jwtMiddleware)
    app.Register(project.NewModule())
    app.Register(http.NewModule(httpPort))

    // Start
    app.Start(context.Background())

    // Print startup info
    printStartupInfo(httpPort, jwtSecret, jwtIssuer)

    // Graceful shutdown
    wait := gfshutdown.GracefulShutdown(
        context.Background(),
        defaultShutdownTimeout,
        map[string]gfshutdown.Operation{
            "mono-app": func(ctx context.Context) error {
                return app.Stop(ctx)
            },
        },
    )

    os.Exit(<-wait)
}

// Helper functions: getEnv, getEnvInt, printStartupInfo
```

**Verification**:
- [ ] Application compiles: `go build ./cmd/static`
- [ ] Application starts successfully
- [ ] Modules register in correct order

---

### Task 4.2: Create Token Generator

**Description**: Implement JWT token generator for testing.

**Files to Create**:
1. `cmd/static/generate_token.go`

**generate_token.go**:
```go
package main

import (
    "fmt"
    "os"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func main() {
    secret := getEnv("JWT_SECRET", "dev-secret-change-in-production")
    issuer := getEnv("JWT_ISSUER", "jwt-example-static")
    userID := getEnv("USER_ID", "user-123")
    email := getEnv("USER_EMAIL", "user@example.com")

    // Create claims
    claims := jwt.MapClaims{
        "iss":   issuer,
        "sub":   userID,
        "email": email,
        "iat":   time.Now().Unix(),
        "exp":   time.Now().Add(1 * time.Hour).Unix(),
    }

    // Sign token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString([]byte(secret))

    // Print output
    fmt.Println("JWT Token Generated")
    fmt.Printf("Token: %s\n", tokenString)
    fmt.Println("\nUsage:")
    fmt.Println("  export JWT_TOKEN=\"" + tokenString + "\"")
    fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" http://localhost:3000/api/v1/projects")
}
```

**Verification**:
- [ ] Generator compiles: `go build ./cmd/static/generate_token.go`
- [ ] Generator produces valid tokens
- [ ] Tokens can be used with the API

---

### Task 4.3: Integration Test

**Description**: Test complete flow with real JWT middleware.

**Files to Create**:
1. `cmd/static/integration_test.go`

**integration_test.go**:
```go
//go:build integration
// +build integration

package main

import (
    "testing"
    // Test full flow: start app, generate token, make HTTP requests
)

func TestStaticSecretIntegration(t *testing.T) {
    // Start application
    // Generate token
    // Test create project (success)
    // Test list projects (success)
    // Test unauthorized (invalid token)
}
```

**Verification**:
- [ ] Integration test passes: `go test -tags=integration ./cmd/static -v`
- [ ] Test creates projects successfully
- [ ] Test rejects invalid tokens

---

## Milestone 5: Secret Provider Strategy

**Goal**: Implement multi-tenant JWT strategy with per-issuer secrets.

### Task 5.1: Implement Secret Store

**Description**: Create thread-safe secret store for multi-tenant secrets.

**Files to Create**:
1. `cmd/secret/main.go`

**Secret Store** (in main.go):
```go
type SecretStore struct {
    mu      sync.RWMutex
    secrets map[string][]byte  // issuer -> secret
}

func NewSecretStore(tenants []TenantSecret) *SecretStore {
    // Initialize with tenant secrets
}

func (s *SecretStore) GetSecret(issuer string) ([]byte, error) {
    // Thread-safe secret lookup
}

func (s *SecretStore) ListIssuers() []string {
    // Return all configured issuers
}
```

**Verification**:
- [ ] Secret store is thread-safe
- [ ] Unknown issuers return error

---

### Task 5.2: Implement Secret Provider Main

**Description**: Create main application for secret provider strategy.

**Files to Create/Update**:
1. `cmd/secret/main.go`

**main.go**:
```go
func main() {
    // Load tenant secrets from env
    tenants := loadTenantSecrets()
    secretStore := NewSecretStore(tenants)

    // Create JWT middleware with secret provider
    jwtMiddleware, _ := jwt.New(
        jwt.WithSecretProvider(secretStore.GetSecret),
    )

    // Register modules and start (similar to static)
}

func loadTenantSecrets() []TenantSecret {
    // Load TENANT_1_NAME, TENANT_1_SECRET, etc.
}
```

**Verification**:
- [ ] Application compiles: `go build ./cmd/secret`
- [ ] Application loads tenant secrets correctly
- [ ] Application starts on port 3001

---

### Task 5.3: Create Multi-Tenant Token Generator

**Description**: Implement token generator with tenant selection.

**Files to Create**:
1. `cmd/secret/generate_token.go`

**generate_token.go**:
```go
func main() {
    tenant := getEnv("TENANT", "tenant-1")
    userID := getEnv("USER_ID", "user-123")

    // Lookup secret for tenant
    secret := getSecretForTenant(tenant)

    // Create token with tenant as issuer
    claims := jwt.MapClaims{
        "iss": tenant,
        "sub": userID,
        // ...
    }

    // Sign and print
}
```

**Verification**:
- [ ] Generator supports TENANT env var
- [ ] Generates tokens for different tenants
- [ ] Tokens have correct issuer claim

---

### Task 5.4: Integration Test

**Description**: Test tenant isolation.

**Files to Create**:
1. `cmd/secret/integration_test.go`

**integration_test.go**:
```go
func TestSecretProviderIntegration(t *testing.T) {
    // Start app
    // Generate tokens for tenant-1 and tenant-2
    // Create projects for each tenant
    // Verify tenant-1 can't see tenant-2's projects
}
```

**Verification**:
- [ ] Integration test passes
- [ ] Tenant isolation works correctly

---

## Milestone 6: JWKS Strategy

**Goal**: Implement RSA/JWKS JWT strategy with mock JWKS server.

### Task 6.1: Implement Mock JWKS Server

**Description**: Create mock JWKS server for demonstration.

**Files to Create**:
1. `cmd/jwks/mock_jwks_server.go`

**mock_jwks_server.go**:
```go
type MockJWKSServer struct {
    port       int
    privateKey *rsa.PrivateKey
    publicKey  *rsa.PublicKey
    keyID      string
    server     *http.Server
}

func NewMockJWKSServer(port int) *MockJWKSServer {
    // Generate RSA key pair
    privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
    return &MockJWKSServer{
        port:       port,
        privateKey: privateKey,
        publicKey:  &privateKey.PublicKey,
        keyID:      "mock-key-1",
    }
}

func (m *MockJWKSServer) Start() error {
    // Start HTTP server with /.well-known/jwks.json endpoint
}

func (m *MockJWKSServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
    // Convert public key to JWK format and return
}
```

**Verification**:
- [ ] Mock server generates valid RSA keys
- [ ] JWKS endpoint returns valid JWK format
- [ ] Public key can validate tokens signed with private key

---

### Task 6.2: Implement JWKS Main

**Description**: Create main application for JWKS strategy.

**Files to Create**:
1. `cmd/jwks/main.go`

**main.go**:
```go
func main() {
    // Check if mock JWKS should be used
    useMockJWKS := getEnvBool("USE_MOCK_JWKS", false)

    var mockServer *MockJWKSServer
    if useMockJWKS || jwksURL == "" {
        mockServer = NewMockJWKSServer(9000)
        mockServer.Start()
        jwksURL = mockServer.JWKSURL()
    }

    // Create JWT middleware with JWKS
    jwtMiddleware, _ := jwt.New(
        jwt.WithJWKS(jwksURL),
        jwt.WithIssuer(issuer),
        jwt.WithAudience(audience),
    )

    // Register modules and start
}
```

**Verification**:
- [ ] Application compiles: `go build ./cmd/jwks`
- [ ] Mock JWKS server starts successfully
- [ ] Application connects to JWKS endpoint
- [ ] Application starts on port 3002

---

### Task 6.3: Create JWKS Token Generator

**Description**: Implement token generator using mock server's private key.

**Files to Create**:
1. `cmd/jwks/generate_token.go`

**generate_token.go**:
```go
func main() {
    // Start mock server to get keys
    mockServer := NewMockJWKSServer(9000)
    mockServer.Start()
    defer mockServer.Stop()

    // Create claims
    claims := jwt.MapClaims{
        "iss": mockServer.Issuer(),
        "aud": mockServer.Audience(),
        "sub": userID,
        // ...
    }

    // Sign with RS256 using private key
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = mockServer.KeyID()
    tokenString, _ := token.SignedString(mockServer.PrivateKey())

    // Print token
}
```

**Verification**:
- [ ] Generator uses RS256 algorithm
- [ ] Token includes kid header
- [ ] Generated tokens validate with JWKS endpoint

---

### Task 6.4: Integration Test

**Description**: Test JWKS validation flow.

**Files to Create**:
1. `cmd/jwks/integration_test.go`

**integration_test.go**:
```go
func TestJWKSIntegration(t *testing.T) {
    // Start app with mock JWKS
    // Generate RS256 token
    // Test create project (success)
    // Test with invalid kid (failure)
}
```

**Verification**:
- [ ] Integration test passes
- [ ] Public key validation works
- [ ] Invalid kid returns 401

---

## Milestone 7: Build System and Tooling

**Goal**: Create Makefile and finalize development tooling.

### Task 7.1: Create Makefile

**Description**: Implement comprehensive Makefile.

**Files to Create**:
1. `Makefile`

**Targets to Implement**:
```makefile
# Build targets
build: build-static build-secret build-jwks
build-static:
build-secret:
build-jwks:

# Run targets
run-static:
run-secret:
run-jwks:

# Token generation
token-static:
token-secret:
token-jwks:

# Testing
test:
test-unit:
test-integration:
test-coverage:

# Code quality
fmt:
lint:
tidy:
check:

# Cleanup
clean:

# Help
help:
```

**Verification**:
- [ ] All build targets work
- [ ] All run targets work
- [ ] All test targets work
- [ ] Help command shows all targets

---

### Task 7.2: Create README

**Description**: Write comprehensive usage documentation.

**Files to Create**:
1. `README.md`

**Sections to Include**:
- Project overview
- Prerequisites
- Quick start for each strategy
- Example curl commands
- Environment variables
- Testing instructions
- When to use each strategy

**Verification**:
- [ ] README has all sections
- [ ] All curl examples work
- [ ] Quick start steps are accurate

---

### Task 7.3: Add Go Module Documentation

**Description**: Add package-level documentation.

**Files to Create**:
1. `doc.go` (package root)
2. `domain/project/doc.go`
3. `modules/project/doc.go`
4. `modules/http/doc.go`

**Verification**:
- [ ] All packages have godoc comments
- [ ] `go doc` shows package documentation

---

## Milestone 8: Final Testing and Verification

**Goal**: Comprehensive testing and quality assurance.

### Task 8.1: Run All Tests

**Description**: Verify all tests pass.

**Commands**:
```bash
make test-unit
make test-integration
make test-coverage
```

**Verification**:
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Coverage meets requirements:
  - Repository: 100%
  - Services: ≥ 90%
  - HTTP handlers: ≥ 85%

---

### Task 8.2: Manual Testing

**Description**: Test each example manually.

**Static Secret**:
```bash
make run-static
make token-static
# Copy token and test API
curl -X POST http://localhost:3000/api/v1/projects \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"Test Project"}'
```

**Secret Provider**:
```bash
make run-secret
make tokens-secret
# Test with different tenant tokens
```

**JWKS**:
```bash
make run-jwks
make token-jwks
# Test RS256 validation
```

**Verification**:
- [ ] All endpoints respond correctly
- [ ] Authentication works for all strategies
- [ ] Authorization prevents cross-user access
- [ ] Error messages are clear

---

### Task 8.3: Code Quality Checks

**Description**: Run code quality tools.

**Commands**:
```bash
make fmt
make lint
make tidy
go vet ./...
```

**Verification**:
- [ ] Code is formatted with gofmt
- [ ] No linter errors (if golangci-lint installed)
- [ ] go mod tidy makes no changes
- [ ] go vet reports no issues

---

### Task 8.4: Cross-Platform Testing

**Description**: Test on Linux and macOS.

**Verification**:
- [ ] Builds on Linux
- [ ] Builds on macOS
- [ ] Tests pass on both platforms
- [ ] Makefile works on both platforms

---

## Success Criteria

All milestones complete when:

- ✅ All code compiles without errors
- ✅ All tests pass (unit + integration)
- ✅ Code coverage meets requirements (100%/90%/85%)
- ✅ All three JWT strategies work correctly
- ✅ Makefile provides all documented commands
- ✅ README is complete with working examples
- ✅ Token generators work for all strategies
- ✅ Manual testing confirms all requirements met
- ✅ Code quality checks pass
- ✅ Cross-platform compatibility verified

---

*Last Updated: January 2026*
*Version: 1.0*
