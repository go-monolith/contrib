package project

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

// ProjectPort defines the interface for project operations.
// This port allows the HTTP module to interact with the project service
// without knowing the internal implementation details (hexagonal architecture).
type ProjectPort interface {
	Create(ctx context.Context, req domain.CreateProjectRequest) (domain.ProjectResponse, error)
	Get(ctx context.Context, req domain.GetProjectRequest) (domain.ProjectResponse, error)
	List(ctx context.Context, req domain.ListProjectsRequest) (domain.ListProjectsResponse, error)
	Update(ctx context.Context, req domain.UpdateProjectRequest) (domain.ProjectResponse, error)
	Delete(ctx context.Context, req domain.DeleteProjectRequest) (domain.ProjectResponse, error)
}

// Adapter implements ProjectPort by calling project services via the service container.
// It uses Mono's RequestReplyService pattern for inter-module communication.
type Adapter struct {
	container mono.ServiceContainer
}

// NewAdapter creates a new project adapter.
func NewAdapter(container mono.ServiceContainer) *Adapter {
	return &Adapter{
		container: container,
	}
}

// Create creates a new project via the project service.
func (a *Adapter) Create(ctx context.Context, req domain.CreateProjectRequest) (domain.ProjectResponse, error) {
	client, err := a.container.GetRequestReplyService("create")
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to get create service: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, data)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	var result domain.ProjectResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Get retrieves a project by ID via the project service.
func (a *Adapter) Get(ctx context.Context, req domain.GetProjectRequest) (domain.ProjectResponse, error) {
	client, err := a.container.GetRequestReplyService("get")
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to get service: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, data)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	var result domain.ProjectResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// List retrieves all projects for the authenticated user via the project service.
func (a *Adapter) List(ctx context.Context, req domain.ListProjectsRequest) (domain.ListProjectsResponse, error) {
	client, err := a.container.GetRequestReplyService("list")
	if err != nil {
		return domain.ListProjectsResponse{}, fmt.Errorf("failed to get list service: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return domain.ListProjectsResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, data)
	if err != nil {
		return domain.ListProjectsResponse{}, err
	}

	var result domain.ListProjectsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return domain.ListProjectsResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Update updates a project via the project service.
func (a *Adapter) Update(ctx context.Context, req domain.UpdateProjectRequest) (domain.ProjectResponse, error) {
	client, err := a.container.GetRequestReplyService("update")
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to get update service: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, data)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	var result domain.ProjectResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Delete deletes a project via the project service.
func (a *Adapter) Delete(ctx context.Context, req domain.DeleteProjectRequest) (domain.ProjectResponse, error) {
	client, err := a.container.GetRequestReplyService("delete")
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to get delete service: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, data)
	if err != nil {
		return domain.ProjectResponse{}, err
	}

	var result domain.ProjectResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return domain.ProjectResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
