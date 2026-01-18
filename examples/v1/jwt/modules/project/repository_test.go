package project

import (
	"fmt"
	"sync"
	"testing"
	"time"

	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create(t *testing.T) {
	repo := NewRepository()
	project := &domain.Project{
		ID:          "proj-123",
		Name:        "Test Project",
		Description: "A test project",
		OwnerID:     "user-123",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(project)
	require.NoError(t, err)

	// Verify project was added
	found, err := repo.FindByID("proj-123")
	require.NoError(t, err)
	assert.Equal(t, project.ID, found.ID)
	assert.Equal(t, project.Name, found.Name)
}

func TestRepository_Create_Duplicate(t *testing.T) {
	repo := NewRepository()
	project := &domain.Project{
		ID:      "proj-123",
		Name:    "Test Project",
		OwnerID: "user-123",
	}

	err := repo.Create(project)
	require.NoError(t, err)

	// Try to create duplicate
	err = repo.Create(project)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRepository_FindByID(t *testing.T) {
	repo := NewRepository()
	project := &domain.Project{
		ID:      "proj-123",
		Name:    "Test Project",
		OwnerID: "user-123",
	}

	err := repo.Create(project)
	require.NoError(t, err)

	found, err := repo.FindByID("proj-123")
	require.NoError(t, err)
	assert.Equal(t, "proj-123", found.ID)
	assert.Equal(t, "Test Project", found.Name)
}

func TestRepository_FindByID_NotFound(t *testing.T) {
	repo := NewRepository()

	found, err := repo.FindByID("non-existent")
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_FindByOwner(t *testing.T) {
	repo := NewRepository()

	// Create multiple projects for same owner
	projects := []*domain.Project{
		{ID: "proj-1", Name: "Project 1", OwnerID: "user-123"},
		{ID: "proj-2", Name: "Project 2", OwnerID: "user-123"},
		{ID: "proj-3", Name: "Project 3", OwnerID: "user-456"},
	}

	for _, p := range projects {
		err := repo.Create(p)
		require.NoError(t, err)
	}

	// Find projects for user-123
	found, err := repo.FindByOwner("user-123")
	require.NoError(t, err)
	assert.Len(t, found, 2)

	// Verify correct projects returned
	ids := make([]string, len(found))
	for i, p := range found {
		ids[i] = p.ID
	}
	assert.Contains(t, ids, "proj-1")
	assert.Contains(t, ids, "proj-2")
	assert.NotContains(t, ids, "proj-3")
}

func TestRepository_FindByOwner_Empty(t *testing.T) {
	repo := NewRepository()

	found, err := repo.FindByOwner("non-existent-user")
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestRepository_ExistsByOwnerAndName(t *testing.T) {
	repo := NewRepository()

	project := &domain.Project{
		ID:      "proj-123",
		Name:    "Test Project",
		OwnerID: "user-123",
	}

	err := repo.Create(project)
	require.NoError(t, err)

	// Check if exists
	exists, err := repo.ExistsByOwnerAndName("user-123", "Test Project")
	require.NoError(t, err)
	assert.True(t, exists)

	// Check different name
	exists, err = repo.ExistsByOwnerAndName("user-123", "Other Project")
	require.NoError(t, err)
	assert.False(t, exists)

	// Check different owner
	exists, err = repo.ExistsByOwnerAndName("user-456", "Test Project")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_Update(t *testing.T) {
	repo := NewRepository()

	project := &domain.Project{
		ID:          "proj-123",
		Name:        "Original Name",
		Description: "Original Description",
		OwnerID:     "user-123",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(project)
	require.NoError(t, err)

	// Update project
	project.Name = "Updated Name"
	project.Description = "Updated Description"
	project.UpdatedAt = time.Now()

	err = repo.Update(project)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID("proj-123")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "Updated Description", found.Description)
}

func TestRepository_Update_NotFound(t *testing.T) {
	repo := NewRepository()

	project := &domain.Project{
		ID:      "non-existent",
		Name:    "Test",
		OwnerID: "user-123",
	}

	err := repo.Update(project)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_Delete(t *testing.T) {
	repo := NewRepository()

	project := &domain.Project{
		ID:      "proj-123",
		Name:    "Test Project",
		OwnerID: "user-123",
	}

	err := repo.Create(project)
	require.NoError(t, err)

	// Delete project
	err = repo.Delete("proj-123")
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID("proj-123")
	assert.Error(t, err)
	assert.Nil(t, found)

	// Verify removed from owner index
	projects, err := repo.FindByOwner("user-123")
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	repo := NewRepository()

	err := repo.Delete("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_ConcurrentAccess(t *testing.T) {
	repo := NewRepository()

	// Number of concurrent goroutines
	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently create projects
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			project := &domain.Project{
				ID:      fmt.Sprintf("proj-%d", index),
				Name:    fmt.Sprintf("Project %d", index),
				OwnerID: fmt.Sprintf("user-%d", index%10), // 10 different owners
			}

			err := repo.Create(project)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all projects were created
	for i := 0; i < numGoroutines; i++ {
		found, err := repo.FindByID(fmt.Sprintf("proj-%d", i))
		assert.NoError(t, err)
		assert.NotNil(t, found)
	}

	// Verify owner indexes are correct
	for i := 0; i < 10; i++ {
		projects, err := repo.FindByOwner(fmt.Sprintf("user-%d", i))
		assert.NoError(t, err)
		assert.Equal(t, 10, len(projects)) // Each owner should have 10 projects
	}
}
