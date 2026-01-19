package project

import (
	"context"
	"encoding/json"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// Module implements the project module as a ServiceProviderModule.
// It provides CRUD services for project management with JWT-based authentication.
type Module struct {
	repo *Repository
}

// Ensure Module implements required interfaces
var _ mono.Module = (*Module)(nil)
var _ mono.ServiceProviderModule = (*Module)(nil)

// NewModule creates a new project module with an in-memory repository.
func NewModule() *Module {
	return &Module{
		repo: NewRepository(),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "project"
}

// Start initializes the module.
// The project module has no startup tasks.
func (m *Module) Start(ctx context.Context) error {
	return nil
}

// Stop cleans up module resources.
// The project module has no cleanup tasks.
func (m *Module) Stop(ctx context.Context) error {
	return nil
}

// RegisterServices registers all project CRUD services with the service container.
// Services are registered with typed handlers for type safety.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register create service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"create",
		json.Unmarshal,
		json.Marshal,
		m.createProject,
	); err != nil {
		return err
	}

	// Register get service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get",
		json.Unmarshal,
		json.Marshal,
		m.getProject,
	); err != nil {
		return err
	}

	// Register list service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"list",
		json.Unmarshal,
		json.Marshal,
		m.listProjects,
	); err != nil {
		return err
	}

	// Register update service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"update",
		json.Unmarshal,
		json.Marshal,
		m.updateProject,
	); err != nil {
		return err
	}

	// Register delete service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"delete",
		json.Unmarshal,
		json.Marshal,
		m.deleteProject,
	); err != nil {
		return err
	}

	return nil
}
