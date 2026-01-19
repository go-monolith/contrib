package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	httpmod "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
)

const (
	defaultHTTPPort        = 3001
	defaultShutdownTimeout = 30 * time.Second
)

// TenantSecret represents a tenant's JWT secret configuration.
type TenantSecret struct {
	Name   string // Tenant name (used as issuer)
	Secret []byte // HMAC secret for this tenant
}

// SecretStore provides thread-safe storage for multi-tenant JWT secrets.
type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string][]byte // issuer -> secret
}

// NewSecretStore creates a new secret store with the given tenant secrets.
func NewSecretStore(tenants []TenantSecret) *SecretStore {
	store := &SecretStore{
		secrets: make(map[string][]byte),
	}

	for _, tenant := range tenants {
		store.secrets[tenant.Name] = tenant.Secret
	}

	return store
}

// GetSecret retrieves the secret for a given issuer (thread-safe).
// This function is used as the secret provider for the JWT middleware.
func (s *SecretStore) GetSecret(issuer string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, ok := s.secrets[issuer]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuer)
	}

	return secret, nil
}

// ListIssuers returns all configured issuers.
func (s *SecretStore) ListIssuers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	issuers := make([]string, 0, len(s.secrets))
	for issuer := range s.secrets {
		issuers = append(issuers, issuer)
	}

	return issuers
}

func main() {
	// Load configuration from environment
	httpPort := getEnvInt("HTTP_PORT", defaultHTTPPort)

	// Load tenant secrets
	tenants := loadTenantSecrets()
	if len(tenants) == 0 {
		log.Fatal("[secret] No tenant secrets configured. Set TENANT_1_NAME and TENANT_1_SECRET environment variables.")
	}

	secretStore := NewSecretStore(tenants)

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(defaultShutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("[secret] Failed to create application: %v", err)
	}

	// Create JWT middleware with secret provider
	jwtMiddleware, err := jwt.New(
		jwt.WithSecretProvider(secretStore.GetSecret),
	)
	if err != nil {
		log.Fatalf("[secret] Failed to create JWT middleware: %v", err)
	}

	// Register modules
	app.Register(jwtMiddleware)
	app.Register(project.NewModule())
	app.Register(httpmod.NewModule(httpPort))

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("[secret] Failed to start application: %v", err)
	}

	// Print startup information
	printStartupInfo(httpPort, secretStore.ListIssuers())

	// Setup graceful shutdown
	wait := gfshutdown.GracefulShutdown(
		context.Background(),
		defaultShutdownTimeout,
		map[string]gfshutdown.Operation{
			"mono-app": func(ctx context.Context) error {
				return app.Stop(ctx)
			},
		},
	)

	os.Exit(<-wait)
}

// loadTenantSecrets loads tenant secrets from environment variables.
// Expects TENANT_N_NAME and TENANT_N_SECRET where N is 1, 2, 3, etc.
func loadTenantSecrets() []TenantSecret {
	var tenants []TenantSecret

	// Try to load up to 10 tenants
	for i := 1; i <= 10; i++ {
		nameKey := fmt.Sprintf("TENANT_%d_NAME", i)
		secretKey := fmt.Sprintf("TENANT_%d_SECRET", i)

		name := os.Getenv(nameKey)
		secret := os.Getenv(secretKey)

		// Stop when we don't find a tenant
		if name == "" || secret == "" {
			break
		}

		tenants = append(tenants, TenantSecret{
			Name:   name,
			Secret: []byte(secret),
		})
	}

	return tenants
}

// getEnv retrieves an environment variable or returns the default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns the default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// printStartupInfo prints application startup information.
func printStartupInfo(port int, issuers []string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("JWT Example - Secret Provider Strategy")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("HTTP Server:    http://localhost:%d\n", port)
	fmt.Printf("Health Check:   http://localhost:%d/health\n", port)
	fmt.Printf("API Endpoint:   http://localhost:%d/api/v1/projects\n", port)
	fmt.Println()
	fmt.Println("Configured Tenants:")
	for i, issuer := range issuers {
		fmt.Printf("  %d. %s\n", i+1, issuer)
	}
	fmt.Println()
	fmt.Println("Strategy: Multi-tenant with per-issuer secrets")
	fmt.Println("Algorithm: HS256")
	fmt.Println()
	fmt.Println("Generate test tokens:")
	fmt.Println("  go run cmd/secret/generate_token.go")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Println("  export TENANT=tenant-1")
	fmt.Println("  export JWT_TOKEN=$(go run cmd/secret/generate_token.go | grep 'Token:' | cut -d' ' -f2)")
	fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" http://localhost:3001/api/v1/projects")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}
