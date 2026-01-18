package http

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Module implements the HTTP module as a DependentModule.
// It provides a REST API for project management via Fiber.
type Module struct {
	app     *fiber.App
	port    int
	project project.ProjectPort
}

// Ensure Module implements required interfaces
var _ mono.Module = (*Module)(nil)
var _ mono.DependentModule = (*Module)(nil)
var _ mono.HealthCheckableModule = (*Module)(nil)

// NewModule creates a new HTTP module that will listen on the specified port.
func NewModule(port int) *Module {
	return &Module{
		port: port,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "http"
}

// Dependencies returns the list of module dependencies.
func (m *Module) Dependencies() []string {
	return []string{"project"}
}

// SetDependencyServiceContainer sets the service container for a dependency.
// This is called by the Mono framework to inject dependencies.
func (m *Module) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
	if dep == "project" {
		m.project = project.NewAdapter(container)
	}
}

// Start initializes and starts the HTTP server.
func (m *Module) Start(ctx context.Context) error {
	// Create Fiber app
	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
	})

	// Add middleware
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	m.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Setup routes
	m.setupRoutes()

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", m.port)
		log.Printf("[http] Starting server on %s", addr)
		if err := m.app.Listen(addr); err != nil {
			log.Printf("[http] Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (m *Module) Stop(ctx context.Context) error {
	if m.app != nil {
		log.Printf("[http] Shutting down server...")
		return m.app.ShutdownWithContext(ctx)
	}
	return nil
}

// Health returns the health status of the HTTP module.
func (m *Module) Health(ctx context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"port": m.port,
		},
	}
}

// setupRoutes configures all HTTP routes.
func (m *Module) setupRoutes() {
	// Health check (no authentication required)
	m.app.Get("/health", m.healthHandler)

	// API v1
	v1 := m.app.Group("/api/v1")

	// Project routes (all require authentication)
	projects := v1.Group("/projects")
	projects.Post("", m.createProjectHandler)
	projects.Get("/:id", m.getProjectHandler)
	projects.Get("", m.listProjectsHandler)
	projects.Put("/:id", m.updateProjectHandler)
	projects.Delete("/:id", m.deleteProjectHandler)
}

// customErrorHandler handles errors from Fiber handlers.
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(ErrorResponse{
		Error:   fiber.ErrInternalServerError.Message,
		Message: err.Error(),
	})
}
