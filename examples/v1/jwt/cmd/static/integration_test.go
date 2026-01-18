//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/contrib/v1/jwt"
	httpmod "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPort      = 3100
	testSecret    = "test-secret-for-integration"
	testIssuer    = "test-issuer"
	baseURL       = "http://localhost:3100"
	healthURL     = baseURL + "/health"
	projectsURL   = baseURL + "/api/v1/projects"
)

// TestStaticSecretIntegration tests the complete flow with JWT middleware
func TestStaticSecretIntegration(t *testing.T) {
	// Start application
	_, cleanup := startTestApp(t)
	defer cleanup()

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(healthURL)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "Server did not start in time")

	// Test 1: Health check (no authentication required)
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(healthURL)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 2: Create project with valid token
	var projectID string
	t.Run("CreateProject_Success", func(t *testing.T) {
		token := generateTestToken("user-123")
		reqBody := domain.CreateProjectRequest{
			Name:        "Integration Test Project",
			Description: "Created during integration test",
		}

		project := createProject(t, token, reqBody)
		assert.NotEmpty(t, project.ID)
		assert.Equal(t, "Integration Test Project", project.Name)
		assert.Equal(t, "user-123", project.OwnerID)

		projectID = project.ID
	})

	// Test 3: Get project with valid token
	t.Run("GetProject_Success", func(t *testing.T) {
		token := generateTestToken("user-123")
		project := getProject(t, token, projectID)
		assert.Equal(t, projectID, project.ID)
		assert.Equal(t, "Integration Test Project", project.Name)
	})

	// Test 4: List projects
	t.Run("ListProjects_Success", func(t *testing.T) {
		token := generateTestToken("user-123")
		listResp := listProjects(t, token)
		assert.GreaterOrEqual(t, listResp.Total, 1)
		assert.GreaterOrEqual(t, len(listResp.Projects), 1)
	})

	// Test 5: Update project
	t.Run("UpdateProject_Success", func(t *testing.T) {
		token := generateTestToken("user-123")
		newName := "Updated Project Name"
		updateReq := domain.UpdateProjectRequest{
			ID:   projectID,
			Name: &newName,
		}
		reqBody, _ := json.Marshal(updateReq)

		req, _ := http.NewRequest(http.MethodPut, projectsURL+"/"+projectID, bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var project domain.ProjectResponse
		json.NewDecoder(resp.Body).Decode(&project)
		assert.Equal(t, "Updated Project Name", project.Name)
	})

	// Test 6: Unauthorized access (no token)
	t.Run("CreateProject_Unauthorized_NoToken", func(t *testing.T) {
		reqBody := domain.CreateProjectRequest{Name: "Should Fail"}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(projectsURL, "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 7: Forbidden access (different user)
	t.Run("GetProject_Forbidden_DifferentUser", func(t *testing.T) {
		token := generateTestToken("user-456") // Different user
		req, _ := http.NewRequest(http.MethodGet, projectsURL+"/"+projectID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test 8: Delete project
	t.Run("DeleteProject_Success", func(t *testing.T) {
		token := generateTestToken("user-123")
		req, _ := http.NewRequest(http.MethodDelete, projectsURL+"/"+projectID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify deletion
		req, _ = http.NewRequest(http.MethodGet, projectsURL+"/"+projectID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// startTestApp starts a test application instance
func startTestApp(t *testing.T) (mono.MonoApplication, func()) {
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(5*time.Second),
	)
	require.NoError(t, err)

	jwtMiddleware, err := jwt.New(
		jwt.WithSecret([]byte(testSecret)),
		jwt.WithExpectedIssuer(testIssuer),
	)
	require.NoError(t, err)

	app.Register(jwtMiddleware)
	app.Register(project.NewModule())
	app.Register(httpmod.NewModule(testPort))

	err = app.Start(context.Background())
	require.NoError(t, err)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		app.Stop(ctx)
	}

	return app, cleanup
}

// generateTestToken generates a JWT token for testing
func generateTestToken(userID string) string {
	claims := jwtgo.MapClaims{
		"iss":   testIssuer,
		"sub":   userID,
		"email": userID + "@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	return tokenString
}

// createProject creates a project via HTTP API
func createProject(t *testing.T, token string, reqBody domain.CreateProjectRequest) domain.ProjectResponse {
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, projectsURL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var project domain.ProjectResponse
	err = json.NewDecoder(resp.Body).Decode(&project)
	require.NoError(t, err)
	return project
}

// getProject retrieves a project via HTTP API
func getProject(t *testing.T, token, projectID string) domain.ProjectResponse {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", projectsURL, projectID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var project domain.ProjectResponse
	err = json.NewDecoder(resp.Body).Decode(&project)
	require.NoError(t, err)
	return project
}

// listProjects lists all projects via HTTP API
func listProjects(t *testing.T, token string) domain.ListProjectsResponse {
	req, _ := http.NewRequest(http.MethodGet, projectsURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp domain.ListProjectsResponse
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	require.NoError(t, err)
	return listResp
}
