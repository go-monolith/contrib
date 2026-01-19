package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	httpmod "github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
)

const (
	defaultHTTPPort        = 3002
	defaultJWKSPort        = 9000
	defaultJWTIssuer       = "mock-jwks-issuer"
	defaultJWTAudience     = "mock-jwks-audience"
	defaultShutdownTimeout = 30 * time.Second
)

func main() {
	// Load configuration from environment
	httpPort := getEnvInt("HTTP_PORT", defaultHTTPPort)
	jwksURL := getEnv("JWKS_URL", "")
	jwtIssuer := getEnv("JWT_ISSUER", defaultJWTIssuer)
	jwtAudience := getEnv("JWT_AUDIENCE", defaultJWTAudience)
	useMockJWKS := getEnvBool("USE_MOCK_JWKS", false)

	// Start mock JWKS server if needed
	var mockServer *MockJWKSServer
	var mockCleanup func(context.Context) error

	if useMockJWKS || jwksURL == "" {
		log.Println("[main] Starting mock JWKS server")
		server, err := NewMockJWKSServer(defaultJWKSPort)
		if err != nil {
			log.Fatalf("[main] Failed to create mock JWKS server: %v", err)
		}

		if err := server.Start(); err != nil {
			log.Fatalf("[main] Failed to start mock JWKS server: %v", err)
		}

		mockServer = server
		jwksURL = server.JWKSURL()
		jwtIssuer = server.Issuer()
		jwtAudience = server.Audience()

		mockCleanup = func(ctx context.Context) error {
			return server.Stop(ctx)
		}
	}

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(defaultShutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("[main] Failed to create application: %v", err)
	}

	// Create JWT middleware with JWKS
	jwtMiddleware, err := jwt.New(
		jwt.WithJWKSEndpoint(jwksURL),
		jwt.WithExpectedIssuer(jwtIssuer),
		jwt.WithExpectedAudience(jwtAudience),
	)
	if err != nil {
		log.Fatalf("[main] Failed to create JWT middleware: %v", err)
	}

	// Register modules
	app.Register(jwtMiddleware)
	app.Register(project.NewModule())
	app.Register(httpmod.NewModule(httpPort))

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("[main] Failed to start application: %v", err)
	}

	// Print startup information
	printStartupInfo(httpPort, jwksURL, jwtIssuer, jwtAudience, mockServer)

	// Setup graceful shutdown
	operations := map[string]gfshutdown.Operation{
		"mono-app": func(ctx context.Context) error {
			return app.Stop(ctx)
		},
	}

	if mockCleanup != nil {
		operations["mock-jwks"] = mockCleanup
	}

	wait := gfshutdown.GracefulShutdown(
		context.Background(),
		defaultShutdownTimeout,
		operations,
	)

	os.Exit(<-wait)
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

// getEnvBool retrieves a boolean environment variable or returns the default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		value = strings.ToLower(value)
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// printStartupInfo prints application startup information.
func printStartupInfo(port int, jwksURL, issuer, audience string, mockServer *MockJWKSServer) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("JWT Example - JWKS Strategy")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("HTTP Server:    http://localhost:%d\n", port)
	fmt.Printf("Health Check:   http://localhost:%d/health\n", port)
	fmt.Printf("API Endpoint:   http://localhost:%d/api/v1/projects\n", port)
	fmt.Println()
	fmt.Println("JWT Configuration:")
	fmt.Printf("  JWKS URL:     %s\n", jwksURL)
	fmt.Printf("  Issuer:       %s\n", issuer)
	fmt.Printf("  Audience:     %s\n", audience)
	fmt.Println()
	fmt.Println("Strategy: JWKS with RSA public keys")
	fmt.Println("Algorithm: RS256")
	fmt.Println()

	if mockServer != nil {
		fmt.Println("Mock JWKS Server:")
		fmt.Printf("  Port:         %d\n", defaultJWKSPort)
		fmt.Printf("  JWKS URL:     %s\n", mockServer.JWKSURL())
		fmt.Printf("  Key ID:       %s\n", mockServer.KeyID())
		fmt.Println()
	}

	fmt.Println("Generate test tokens:")
	fmt.Println("  go run -tags=tokengen cmd/jwks/generate_token.go")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Println("  export JWT_TOKEN=$(go run -tags=tokengen cmd/jwks/generate_token.go | grep 'Token:' | cut -d' ' -f2)")
	fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" http://localhost:3002/api/v1/projects")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}
