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
	testPort    = 3102
	jwksPort    = 9001
	baseURL     = "http://localhost:3102"
	healthURL   = baseURL + "/health"
	projectsURL = baseURL + "/api/v1/projects"
)

// TestJWKSIntegration tests the complete JWKS flow with RSA keys.
func TestJWKSIntegration(t *testing.T) {
	// Start mock JWKS server
	mockServer, err := NewMockJWKSServer(jwksPort)
	require.NoError(t, err)

	err = mockServer.Start()
	require.NoError(t, err)
	defer mockServer.Stop(context.Background())

	// Start application with JWKS
	_, cleanup := startTestApp(t, mockServer)
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

	// Test 2: Create project with valid RS256 token
	var projectID string
	t.Run("CreateProject_RS256_Success", func(t *testing.T) {
		token := generateTestToken(t, mockServer, "user-123")
		reqBody := domain.CreateProjectRequest{
			Name:        "JWKS Test Project",
			Description: "Project with RS256 token",
		}

		project := createProject(t, token, reqBody)
		assert.NotEmpty(t, project.ID)
		assert.Equal(t, "JWKS Test Project", project.Name)
		// OwnerID should be issuer:sub format
		assert.Equal(t, mockServer.Issuer()+":user-123", project.OwnerID)

		projectID = project.ID
	})

	// Test 3: Get project with valid token
	t.Run("GetProject_Success", func(t *testing.T) {
		token := generateTestToken(t, mockServer, "user-123")
		project := getProject(t, token, projectID)
		assert.Equal(t, projectID, project.ID)
		assert.Equal(t, "JWKS Test Project", project.Name)
	})

	// Test 4: List projects
	t.Run("ListProjects_Success", func(t *testing.T) {
		token := generateTestToken(t, mockServer, "user-123")
		listResp := listProjects(t, token)
		assert.GreaterOrEqual(t, listResp.Total, 1)
		assert.GreaterOrEqual(t, len(listResp.Projects), 1)
	})

	// Test 5: Update project
	t.Run("UpdateProject_Success", func(t *testing.T) {
		token := generateTestToken(t, mockServer, "user-123")
		newName := "Updated JWKS Project"
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
		assert.Equal(t, "Updated JWKS Project", project.Name)
	})

	// Test 6: Invalid token (wrong signature)
	t.Run("CreateProject_InvalidSignature_Unauthorized", func(t *testing.T) {
		// Create a token with wrong issuer/signature
		claims := jwtgo.MapClaims{
			"iss":   "wrong-issuer",
			"aud":   mockServer.Audience(),
			"sub":   "user-123",
			"email": "user@example.com",
			"iat":   time.Now().Unix(),
			"exp":   time.Now().Add(1 * time.Hour).Unix(),
		}

		token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
		token.Header["kid"] = mockServer.KeyID()

		// Sign with the same key but wrong claims will still validate signature
		// but fail issuer validation
		tokenString, _ := token.SignedString(mockServer.PrivateKey())

		reqBody := domain.CreateProjectRequest{Name: "Should Fail"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, projectsURL, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 7: Missing kid header
	t.Run("CreateProject_MissingKid_Unauthorized", func(t *testing.T) {
		claims := jwtgo.MapClaims{
			"iss":   mockServer.Issuer(),
			"aud":   mockServer.Audience(),
			"sub":   "user-123",
			"email": "user@example.com",
			"iat":   time.Now().Unix(),
			"exp":   time.Now().Add(1 * time.Hour).Unix(),
		}

		token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
		// Don't set kid header
		tokenString, _ := token.SignedString(mockServer.PrivateKey())

		reqBody := domain.CreateProjectRequest{Name: "Should Fail"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, projectsURL, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 8: Delete project
	t.Run("DeleteProject_Success", func(t *testing.T) {
		token := generateTestToken(t, mockServer, "user-123")
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

// startTestApp starts a test application with JWKS configuration.
func startTestApp(t *testing.T, mockServer *MockJWKSServer) (mono.MonoApplication, func()) {
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(5*time.Second),
	)
	require.NoError(t, err)

	// Create JWT middleware with JWKS
	jwtMiddleware, err := jwt.New(
		jwt.WithJWKSEndpoint(mockServer.JWKSURL()),
		jwt.WithExpectedIssuer(mockServer.Issuer()),
		jwt.WithExpectedAudience(mockServer.Audience()),
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

// generateTestToken generates a valid RS256 JWT token.
func generateTestToken(t *testing.T, server *MockJWKSServer, userID string) string {
	claims := jwtgo.MapClaims{
		"iss":   server.Issuer(),
		"aud":   server.Audience(),
		"sub":   userID,
		"email": userID + "@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
	token.Header["kid"] = server.KeyID()

	tokenString, err := token.SignedString(server.PrivateKey())
	require.NoError(t, err)

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

// getProject retrieves a project via HTTP API.
func getProject(t *testing.T, token, projectID string) domain.ProjectResponse {
	req, _ := http.NewRequest(http.MethodGet, projectsURL+"/"+projectID, nil)
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
