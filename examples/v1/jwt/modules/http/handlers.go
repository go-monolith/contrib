package http

import (
	"context"

	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	"github.com/gofiber/fiber/v2"
)

const authHeaderKey = "jwt-authorization-header"

// contextWithAuth adds the Authorization header from Fiber context to the standard context
func contextWithAuth(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if authHeader := c.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, authHeaderKey, authHeader)
	}
	return ctx
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// healthHandler handles health check requests.
func (m *Module) healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "jwt-example",
	})
}

// createProjectHandler handles POST /api/v1/projects
func (m *Module) createProjectHandler(c *fiber.Ctx) error {
	var req domain.CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "invalid request body",
		})
	}

	resp, err := m.project.Create(contextWithAuth(c), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// getProjectHandler handles GET /api/v1/projects/:id
func (m *Module) getProjectHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "project ID is required",
		})
	}

	req := domain.GetProjectRequest{
		ID: id,
	}

	resp, err := m.project.Get(contextWithAuth(c), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(resp)
}

// listProjectsHandler handles GET /api/v1/projects
func (m *Module) listProjectsHandler(c *fiber.Ctx) error {
	req := domain.ListProjectsRequest{}

	resp, err := m.project.List(contextWithAuth(c), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(resp)
}

// updateProjectHandler handles PUT /api/v1/projects/:id
func (m *Module) updateProjectHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "project ID is required",
		})
	}

	var req domain.UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "invalid request body",
		})
	}

	// Set ID from URL
	req.ID = id

	resp, err := m.project.Update(contextWithAuth(c), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(resp)
}

// deleteProjectHandler handles DELETE /api/v1/projects/:id
func (m *Module) deleteProjectHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "project ID is required",
		})
	}

	req := domain.DeleteProjectRequest{
		ID: id,
	}

	resp, err := m.project.Delete(contextWithAuth(c), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(resp)
}

// handleServiceError maps service errors to HTTP status codes.
func handleServiceError(c *fiber.Ctx, err error) error {
	switch {
	case isUnauthorizedError(err):
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
	case isForbiddenError(err):
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error:   "Forbidden",
			Message: err.Error(),
		})
	case isNotFoundError(err):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	case isValidationError(err):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}
}
