package project

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockContext creates a context with JWT claims for testing
func mockContext(userID string) context.Context {
	claims := jwtgo.MapClaims{
		"sub":   userID,
		"email": userID + "@example.com",
		"iss":   "test-issuer",
	}
	return jwt.WithClaims(context.Background(), claims)
}

// TestCreateProject_Success tests successful project creation
func TestCreateProject_Success(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	req := domain.CreateProjectRequest{
		Name:        "Test Project",
		Description: "A test project",
	}

	resp, err := m.createProject(ctx, req, &mono.Msg{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Test Project", resp.Name)
	assert.Equal(t, "A test project", resp.Description)
	assert.Equal(t, "test-issuer:user-123", resp.OwnerID) // Composite ID: issuer:sub
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

// TestCreateProject_MissingClaims tests creation without JWT claims
func TestCreateProject_MissingClaims(t *testing.T) {
	m := NewModule()
	ctx := context.Background() // No claims

	req := domain.CreateProjectRequest{
		Name: "Test Project",
	}

	resp, err := m.createProject(ctx, req, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no JWT claims")
	assert.Empty(t, resp.ID)
}

// TestCreateProject_EmptyName tests validation for empty name
func TestCreateProject_EmptyName(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	req := domain.CreateProjectRequest{
		Name: "",
	}

	resp, err := m.createProject(ctx, req, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
	assert.Empty(t, resp.ID)
}

// TestCreateProject_NameTooLong tests validation for name length
func TestCreateProject_NameTooLong(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	longName := string(make([]byte, 101))
	req := domain.CreateProjectRequest{
		Name: longName,
	}

	resp, err := m.createProject(ctx, req, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exceed 100 characters")
	assert.Empty(t, resp.ID)
}

// TestCreateProject_DescriptionTooLong tests validation for description length
func TestCreateProject_DescriptionTooLong(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	longDesc := string(make([]byte, 501))
	req := domain.CreateProjectRequest{
		Name:        "Test Project",
		Description: longDesc,
	}

	resp, err := m.createProject(ctx, req, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exceed 500 characters")
	assert.Empty(t, resp.ID)
}

// TestCreateProject_DuplicateName tests duplicate name validation
func TestCreateProject_DuplicateName(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	req := domain.CreateProjectRequest{
		Name: "Test Project",
	}

	// Create first project
	_, err := m.createProject(ctx, req, &mono.Msg{})
	require.NoError(t, err)

	// Try to create duplicate
	resp, err := m.createProject(ctx, req, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Empty(t, resp.ID)
}

// TestGetProject_Success tests successful project retrieval
func TestGetProject_Success(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create project first
	createReq := domain.CreateProjectRequest{
		Name: "Test Project",
	}
	created, err := m.createProject(ctx, createReq, &mono.Msg{})
	require.NoError(t, err)

	// Get project
	getReq := domain.GetProjectRequest{
		ID: created.ID,
	}
	resp, err := m.getProject(ctx, getReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, created.ID, resp.ID)
	assert.Equal(t, "Test Project", resp.Name)
}

// TestGetProject_NotFound tests getting non-existent project
func TestGetProject_NotFound(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	getReq := domain.GetProjectRequest{
		ID: "non-existent",
	}
	resp, err := m.getProject(ctx, getReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, resp.ID)
}

// TestGetProject_Forbidden tests getting another user's project
func TestGetProject_Forbidden(t *testing.T) {
	m := NewModule()
	ctx1 := mockContext("user-123")
	ctx2 := mockContext("user-456")

	// User 1 creates project
	createReq := domain.CreateProjectRequest{
		Name: "User 1 Project",
	}
	created, err := m.createProject(ctx1, createReq, &mono.Msg{})
	require.NoError(t, err)

	// User 2 tries to get it
	getReq := domain.GetProjectRequest{
		ID: created.ID,
	}
	resp, err := m.getProject(ctx2, getReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	assert.Empty(t, resp.ID)
}

// TestListProjects_Success tests listing projects
func TestListProjects_Success(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create multiple projects
	for i := 1; i <= 3; i++ {
		createReq := domain.CreateProjectRequest{
			Name: fmt.Sprintf("Project %d", i),
		}
		_, err := m.createProject(ctx, createReq, &mono.Msg{})
		require.NoError(t, err)
	}

	// List projects
	listReq := domain.ListProjectsRequest{}
	resp, err := m.listProjects(ctx, listReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Len(t, resp.Projects, 3)
}

// TestListProjects_Empty tests listing with no projects
func TestListProjects_Empty(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	listReq := domain.ListProjectsRequest{}
	resp, err := m.listProjects(ctx, listReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Projects)
}

// TestUpdateProject_Success tests successful project update
func TestUpdateProject_Success(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create project
	createReq := domain.CreateProjectRequest{
		Name:        "Original Name",
		Description: "Original Description",
	}
	created, err := m.createProject(ctx, createReq, &mono.Msg{})
	require.NoError(t, err)

	// Update project
	newName := "Updated Name"
	newDesc := "Updated Description"
	updateReq := domain.UpdateProjectRequest{
		ID:          created.ID,
		Name:        &newName,
		Description: &newDesc,
	}
	resp, err := m.updateProject(ctx, updateReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, "Updated Description", resp.Description)
}

// TestUpdateProject_PartialUpdate tests partial update
func TestUpdateProject_PartialUpdate(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create project
	createReq := domain.CreateProjectRequest{
		Name:        "Original Name",
		Description: "Original Description",
	}
	created, err := m.createProject(ctx, createReq, &mono.Msg{})
	require.NoError(t, err)

	// Update only name
	newName := "Updated Name"
	updateReq := domain.UpdateProjectRequest{
		ID:   created.ID,
		Name: &newName,
	}
	resp, err := m.updateProject(ctx, updateReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, "Original Description", resp.Description)
}

// TestUpdateProject_NotFound tests updating non-existent project
func TestUpdateProject_NotFound(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	newName := "New Name"
	updateReq := domain.UpdateProjectRequest{
		ID:   "non-existent",
		Name: &newName,
	}
	resp, err := m.updateProject(ctx, updateReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, resp.ID)
}

// TestUpdateProject_Forbidden tests updating another user's project
func TestUpdateProject_Forbidden(t *testing.T) {
	m := NewModule()
	ctx1 := mockContext("user-123")
	ctx2 := mockContext("user-456")

	// User 1 creates project
	createReq := domain.CreateProjectRequest{
		Name: "User 1 Project",
	}
	created, err := m.createProject(ctx1, createReq, &mono.Msg{})
	require.NoError(t, err)

	// User 2 tries to update it
	newName := "Hacked Name"
	updateReq := domain.UpdateProjectRequest{
		ID:   created.ID,
		Name: &newName,
	}
	resp, err := m.updateProject(ctx2, updateReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	assert.Empty(t, resp.ID)
}

// TestUpdateProject_DuplicateName tests duplicate name validation on update
func TestUpdateProject_DuplicateName(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create two projects
	createReq1 := domain.CreateProjectRequest{Name: "Project 1"}
	created1, err := m.createProject(ctx, createReq1, &mono.Msg{})
	require.NoError(t, err)

	createReq2 := domain.CreateProjectRequest{Name: "Project 2"}
	created2, err := m.createProject(ctx, createReq2, &mono.Msg{})
	require.NoError(t, err)

	// Try to update Project 2 to have same name as Project 1
	newName := "Project 1"
	updateReq := domain.UpdateProjectRequest{
		ID:   created2.ID,
		Name: &newName,
	}
	resp, err := m.updateProject(ctx, updateReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Empty(t, resp.ID)

	// Verify Project 1 is unchanged
	getReq := domain.GetProjectRequest{ID: created1.ID}
	project1, err := m.getProject(ctx, getReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, "Project 1", project1.Name)
}

// TestUpdateProject_NoFields tests update with no fields
func TestUpdateProject_NoFields(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create project
	createReq := domain.CreateProjectRequest{Name: "Test Project"}
	created, err := m.createProject(ctx, createReq, &mono.Msg{})
	require.NoError(t, err)

	// Try to update without any fields
	updateReq := domain.UpdateProjectRequest{
		ID: created.ID,
	}
	resp, err := m.updateProject(ctx, updateReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field")
	assert.Empty(t, resp.ID)
}

// TestDeleteProject_Success tests successful project deletion
func TestDeleteProject_Success(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	// Create project
	createReq := domain.CreateProjectRequest{Name: "Test Project"}
	created, err := m.createProject(ctx, createReq, &mono.Msg{})
	require.NoError(t, err)

	// Delete project
	deleteReq := domain.DeleteProjectRequest{ID: created.ID}
	resp, err := m.deleteProject(ctx, deleteReq, &mono.Msg{})
	require.NoError(t, err)
	assert.Equal(t, created.ID, resp.ID)

	// Verify deletion
	getReq := domain.GetProjectRequest{ID: created.ID}
	_, err = m.getProject(ctx, getReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestDeleteProject_NotFound tests deleting non-existent project
func TestDeleteProject_NotFound(t *testing.T) {
	m := NewModule()
	ctx := mockContext("user-123")

	deleteReq := domain.DeleteProjectRequest{ID: "non-existent"}
	resp, err := m.deleteProject(ctx, deleteReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, resp.ID)
}

// TestDeleteProject_Forbidden tests deleting another user's project
func TestDeleteProject_Forbidden(t *testing.T) {
	m := NewModule()
	ctx1 := mockContext("user-123")
	ctx2 := mockContext("user-456")

	// User 1 creates project
	createReq := domain.CreateProjectRequest{Name: "User 1 Project"}
	created, err := m.createProject(ctx1, createReq, &mono.Msg{})
	require.NoError(t, err)

	// User 2 tries to delete it
	deleteReq := domain.DeleteProjectRequest{ID: created.ID}
	resp, err := m.deleteProject(ctx2, deleteReq, &mono.Msg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	assert.Empty(t, resp.ID)

	// Verify project still exists
	getReq := domain.GetProjectRequest{ID: created.ID}
	_, err = m.getProject(ctx1, getReq, &mono.Msg{})
	require.NoError(t, err)
}
