package project

import (
	"fmt"
	"sync"

	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
)

// Repository provides thread-safe in-memory storage for projects.
// This is a mock repository for demonstration purposes only.
type Repository struct {
	mu       sync.RWMutex
	projects map[string]*domain.Project // ID → Project
	byOwner  map[string][]string        // OwnerID → []ProjectID
}

// NewRepository creates a new empty repository.
func NewRepository() *Repository {
	return &Repository{
		projects: make(map[string]*domain.Project),
		byOwner:  make(map[string][]string),
	}
}

// Create adds a new project to the repository.
// Returns an error if a project with the same ID already exists.
func (r *Repository) Create(project *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.projects[project.ID]; exists {
		return fmt.Errorf("project with ID %s already exists", project.ID)
	}

	r.projects[project.ID] = project
	r.byOwner[project.OwnerID] = append(r.byOwner[project.OwnerID], project.ID)

	return nil
}

// FindByID retrieves a project by its ID.
// Returns an error if the project is not found.
func (r *Repository) FindByID(id string) (*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.projects[id]
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	return project, nil
}

// FindByOwner retrieves all projects owned by a specific user.
// Returns an empty slice if the user has no projects.
func (r *Repository) FindByOwner(ownerID string) ([]*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectIDs, exists := r.byOwner[ownerID]
	if !exists {
		return []*domain.Project{}, nil
	}

	projects := make([]*domain.Project, 0, len(projectIDs))
	for _, id := range projectIDs {
		if project, ok := r.projects[id]; ok {
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// ExistsByOwnerAndName checks if a project with the given name exists for a specific owner.
// This is used to enforce unique project names per user.
func (r *Repository) ExistsByOwnerAndName(ownerID, name string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectIDs, exists := r.byOwner[ownerID]
	if !exists {
		return false, nil
	}

	for _, id := range projectIDs {
		if project, ok := r.projects[id]; ok && project.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// Update updates an existing project in the repository.
// Returns an error if the project is not found.
func (r *Repository) Update(project *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.projects[project.ID]; !exists {
		return fmt.Errorf("project not found")
	}

	r.projects[project.ID] = project

	return nil
}

// Delete removes a project from the repository.
// Returns an error if the project is not found.
func (r *Repository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.projects[id]
	if !exists {
		return fmt.Errorf("project not found")
	}

	// Remove from projects map
	delete(r.projects, id)

	// Remove from byOwner index
	if projectIDs, ok := r.byOwner[project.OwnerID]; ok {
		for i, pid := range projectIDs {
			if pid == id {
				r.byOwner[project.OwnerID] = append(projectIDs[:i], projectIDs[i+1:]...)
				break
			}
		}
	}

	return nil
}
