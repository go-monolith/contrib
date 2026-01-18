//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	domain "github.com/go-monolith/mono/contrib/examples/v1/jwt/domain/project"
	httpmod "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPort      = 3101
	tenant1Name   = "tenant-1"
	tenant1Secret = "tenant-1-secret"
	tenant2Name   = "tenant-2"
	tenant2Secret = "tenant-2-secret"
	baseURL       = "http://localhost:3101"
	healthURL     = baseURL + "/health"
	projectsURL   = baseURL + "/api/v1/projects"
)

// TestSecretProviderIntegration tests multi-tenant JWT authentication with tenant isolation.
func TestSecretProviderIntegration(t *testing.T) {
	// Start application with multiple tenants
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

	// Test 1: Health check
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(healthURL)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 2: Create project for tenant-1
	var tenant1ProjectID string
	t.Run("CreateProject_Tenant1_Success", func(t *testing.T) {
		token := generateTestToken(tenant1Name, tenant1Secret, "user-123")
		reqBody := domain.CreateProjectRequest{
			Name:        "Tenant 1 Project",
			Description: "Project for tenant-1",
		}

		project := createProject(t, token, reqBody)
		assert.NotEmpty(t, project.ID)
		assert.Equal(t, "Tenant 1 Project", project.Name)
		assert.Equal(t, "tenant-1:user-123", project.OwnerID)

		tenant1ProjectID = project.ID
	})

	// Test 3: Create project for tenant-2
	var tenant2ProjectID string
	t.Run("CreateProject_Tenant2_Success", func(t *testing.T) {
		token := generateTestToken(tenant2Name, tenant2Secret, "user-456")
		reqBody := domain.CreateProjectRequest{
			Name:        "Tenant 2 Project",
			Description: "Project for tenant-2",
		}

		project := createProject(t, token, reqBody)
		assert.NotEmpty(t, project.ID)
		assert.Equal(t, "Tenant 2 Project", project.Name)
		assert.Equal(t, "tenant-2:user-456", project.OwnerID)

		tenant2ProjectID = project.ID
	})

	// Test 4: Tenant-1 can list their own projects
	t.Run("ListProjects_Tenant1_SeesOnlyOwnProjects", func(t *testing.T) {
		token := generateTestToken(tenant1Name, tenant1Secret, "user-123")
		listResp := listProjects(t, token)

		assert.Equal(t, 1, listResp.Total)
		assert.Equal(t, 1, len(listResp.Projects))
		assert.Equal(t, "Tenant 1 Project", listResp.Projects[0].Name)
		assert.Equal(t, "tenant-1:user-123", listResp.Projects[0].OwnerID)
	})

	// Test 5: Tenant-2 can list their own projects
	t.Run("ListProjects_Tenant2_SeesOnlyOwnProjects", func(t *testing.T) {
		token := generateTestToken(tenant2Name, tenant2Secret, "user-456")
		listResp := listProjects(t, token)

		assert.Equal(t, 1, listResp.Total)
		assert.Equal(t, 1, len(listResp.Projects))
		assert.Equal(t, "Tenant 2 Project", listResp.Projects[0].Name)
		assert.Equal(t, "tenant-2:user-456", listResp.Projects[0].OwnerID)
	})

	// Test 6: Tenant-2 cannot access tenant-1's project (forbidden)
	t.Run("GetProject_Tenant2_CannotAccessTenant1Project", func(t *testing.T) {
		token := generateTestToken(tenant2Name, tenant2Secret, "user-456")
		req, _ := http.NewRequest(http.MethodGet, projectsURL+"/"+tenant1ProjectID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test 7: Tenant-1 cannot access tenant-2's project (forbidden)
	t.Run("GetProject_Tenant1_CannotAccessTenant2Project", func(t *testing.T) {
		token := generateTestToken(tenant1Name, tenant1Secret, "user-123")
		req, _ := http.NewRequest(http.MethodGet, projectsURL+"/"+tenant2ProjectID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test 8: Invalid token (wrong secret) is rejected
	t.Run("CreateProject_InvalidSecret_Unauthorized", func(t *testing.T) {
		// Use wrong secret for tenant-1
		token := generateTestToken(tenant1Name, "wrong-secret", "user-123")
		reqBody := domain.CreateProjectRequest{Name: "Should Fail"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, projectsURL, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 9: Unknown issuer is rejected
	t.Run("CreateProject_UnknownIssuer_Unauthorized", func(t *testing.T) {
		token := generateTestToken("unknown-tenant", "some-secret", "user-789")
		reqBody := domain.CreateProjectRequest{Name: "Should Fail"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, projectsURL, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 10: Same user in different tenants have isolated data
	t.Run("SameUser_DifferentTenants_IsolatedData", func(t *testing.T) {
		sameUserID := "shared-user-999"

		// Create project for same user in tenant-1
		token1 := generateTestToken(tenant1Name, tenant1Secret, sameUserID)
		reqBody1 := domain.CreateProjectRequest{
			Name: "Project in Tenant 1",
		}
		createProject(t, token1, reqBody1)

		// Create project for same user in tenant-2
		token2 := generateTestToken(tenant2Name, tenant2Secret, sameUserID)
		reqBody2 := domain.CreateProjectRequest{
			Name: "Project in Tenant 2",
		}
		createProject(t, token2, reqBody2)

		// User in tenant-1 should only see tenant-1 project
		list1 := listProjects(t, token1)
		assert.Equal(t, 1, len(list1.Projects))
		assert.Equal(t, "Project in Tenant 1", list1.Projects[0].Name)

		// User in tenant-2 should only see tenant-2 project
		list2 := listProjects(t, token2)
		assert.Equal(t, 1, len(list2.Projects))
		assert.Equal(t, "Project in Tenant 2", list2.Projects[0].Name)
	})
}

// startTestApp starts a test application with multi-tenant configuration.
func startTestApp(t *testing.T) (mono.MonoApplication, func()) {
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(5 * time.Second),
	)
	require.NoError(t, err)

	// Setup multi-tenant secrets
	tenants := []TenantSecret{
		{Name: tenant1Name, Secret: []byte(tenant1Secret)},
		{Name: tenant2Name, Secret: []byte(tenant2Secret)},
	}
	secretStore := NewSecretStore(tenants)

	// Create JWT middleware with secret provider
	jwtMiddleware, err := jwt.New(
		jwt.WithSecretProvider(secretStore.GetSecret),
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

// generateTestToken generates a JWT token for a specific tenant.
func generateTestToken(issuer, secret, userID string) string {
	claims := jwtgo.MapClaims{
		"iss":   issuer,
		"sub":   userID,
		"email": userID + "@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// createProject creates a project via HTTP API.
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

// listProjects lists all projects via HTTP API.
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
