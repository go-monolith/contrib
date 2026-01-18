package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	"github.com/google/uuid"
)

// getUserIDFromContext extracts the user ID from JWT claims in the context.
// For multi-tenant scenarios, it creates a composite ID in the format "issuer:sub"
// to ensure tenant isolation. If no issuer is present, it uses just the sub claim.
// Returns an error if claims are missing or the 'sub' claim is not present.
func getUserIDFromContext(ctx context.Context) (string, error) {
	claims, ok := jwt.ClaimsFromContext(ctx)
	if !ok {
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

	if userID == "" {
		return "", fmt.Errorf("'sub' claim is empty")
	}

	// For multi-tenant isolation, include issuer in the user identity
	if iss, ok := claims["iss"]; ok {
		if issuer, ok := iss.(string); ok && issuer != "" {
			// Create composite ID: issuer:sub
			return fmt.Sprintf("%s:%s", issuer, userID), nil
		}
	}

	// Fallback to just sub if no issuer
	return userID, nil
}

// toProjectResponse converts a project entity to a response DTO.
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

// createProject creates a new project for the authenticated user.
func (m *Module) createProject(
	ctx context.Context,
	req domain.CreateProjectRequest,
	msg *mono.Msg,
) (domain.ProjectResponse, error) {
	// Extract user ID from JWT claims
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Validate request
	if err := m.validateCreateRequest(req); err != nil {
		return domain.ProjectResponse{}, err
	}

	// Check if project with same name already exists for this user
	exists, err := m.repo.ExistsByOwnerAndName(userID, req.Name)
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to check project existence: %w", err)
	}
	if exists {
		return domain.ProjectResponse{}, fmt.Errorf("project with name '%s' already exists", req.Name)
	}

	// Create project entity
	now := time.Now()
	project := &domain.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to repository
	if err := m.repo.Create(project); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to create project: %w", err)
	}

	return toProjectResponse(project), nil
}

// getProject retrieves a project by ID.
// Only the project owner can access the project.
func (m *Module) getProject(
	ctx context.Context,
	req domain.GetProjectRequest,
	msg *mono.Msg,
) (domain.ProjectResponse, error) {
	// Extract user ID from JWT claims
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Validate request
	if req.ID == "" {
		return domain.ProjectResponse{}, fmt.Errorf("project ID is required")
	}

	// Find project
	project, err := m.repo.FindByID(req.ID)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Check authorization (owner-based access control)
	if project.OwnerID != userID {
		return domain.ProjectResponse{}, fmt.Errorf("forbidden: you do not own this project")
	}

	return toProjectResponse(project), nil
}

// listProjects lists all projects owned by the authenticated user.
func (m *Module) listProjects(
	ctx context.Context,
	req domain.ListProjectsRequest,
	msg *mono.Msg,
) (domain.ListProjectsResponse, error) {
	// Extract user ID from JWT claims
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return domain.ListProjectsResponse{}, err
	}

	// Find all projects for this user
	projects, err := m.repo.FindByOwner(userID)
	if err != nil {
		return domain.ListProjectsResponse{}, fmt.Errorf("failed to list projects: %w", err)
	}

	// Convert to response DTOs
	responses := make([]domain.ProjectResponse, len(projects))
	for i, p := range projects {
		responses[i] = toProjectResponse(p)
	}

	return domain.ListProjectsResponse{
		Projects: responses,
		Total:    len(responses),
	}, nil
}

// updateProject updates an existing project.
// Only the project owner can update the project.
// Supports partial updates - only provided fields are updated.
func (m *Module) updateProject(
	ctx context.Context,
	req domain.UpdateProjectRequest,
	msg *mono.Msg,
) (domain.ProjectResponse, error) {
	// Extract user ID from JWT claims
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Validate request
	if req.ID == "" {
		return domain.ProjectResponse{}, fmt.Errorf("project ID is required")
	}

	// Check that at least one field is provided
	if req.Name == nil && req.Description == nil {
		return domain.ProjectResponse{}, fmt.Errorf("at least one field must be provided for update")
	}

	// Find existing project
	project, err := m.repo.FindByID(req.ID)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Check authorization
	if project.OwnerID != userID {
		return domain.ProjectResponse{}, fmt.Errorf("forbidden: you do not own this project")
	}

	// Update fields if provided
	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return domain.ProjectResponse{}, fmt.Errorf("name cannot be empty")
		}
		if len(newName) > 100 {
			return domain.ProjectResponse{}, fmt.Errorf("name cannot exceed 100 characters")
		}

		// Check if new name already exists for this user (excluding current project)
		if newName != project.Name {
			exists, err := m.repo.ExistsByOwnerAndName(userID, newName)
			if err != nil {
				return domain.ProjectResponse{}, fmt.Errorf("failed to check project existence: %w", err)
			}
			if exists {
				return domain.ProjectResponse{}, fmt.Errorf("project with name '%s' already exists", newName)
			}
		}

		project.Name = newName
	}

	if req.Description != nil {
		description := *req.Description
		if len(description) > 500 {
			return domain.ProjectResponse{}, fmt.Errorf("description cannot exceed 500 characters")
		}
		project.Description = description
	}

	// Update timestamp
	project.UpdatedAt = time.Now()

	// Save changes
	if err := m.repo.Update(project); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to update project: %w", err)
	}

	return toProjectResponse(project), nil
}

// deleteProject deletes a project by ID.
// Only the project owner can delete the project.
func (m *Module) deleteProject(
	ctx context.Context,
	req domain.DeleteProjectRequest,
	msg *mono.Msg,
) (domain.ProjectResponse, error) {
	// Extract user ID from JWT claims
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Validate request
	if req.ID == "" {
		return domain.ProjectResponse{}, fmt.Errorf("project ID is required")
	}

	// Find project
	project, err := m.repo.FindByID(req.ID)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	// Check authorization
	if project.OwnerID != userID {
		return domain.ProjectResponse{}, fmt.Errorf("forbidden: you do not own this project")
	}

	// Delete project
	if err := m.repo.Delete(req.ID); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to delete project: %w", err)
	}

	return toProjectResponse(project), nil
}

// validateCreateRequest validates the create project request.
func (m *Module) validateCreateRequest(req domain.CreateProjectRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}

	if len(name) > 100 {
		return fmt.Errorf("name cannot exceed 100 characters")
	}

	if len(req.Description) > 500 {
		return fmt.Errorf("description cannot exceed 500 characters")
	}

	return nil
}
