package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/contrib/v1/jwt"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/http"
	"github.com/go-monolith/mono/contrib/examples/v1/jwt/modules/project"
)

const (
	defaultHTTPPort        = 3000
	defaultJWTSecret       = "dev-secret-change-in-production"
	defaultJWTIssuer       = "jwt-example-static"
	defaultShutdownTimeout = 30 * time.Second
)

func main() {
	// Load configuration from environment
	httpPort := getEnvInt("HTTP_PORT", defaultHTTPPort)
	jwtSecret := getEnv("JWT_SECRET", defaultJWTSecret)
	jwtIssuer := getEnv("JWT_ISSUER", defaultJWTIssuer)

	// Security warning for default secret
	if jwtSecret == defaultJWTSecret {
		log.Println("⚠️  WARNING: Using default JWT secret! This is insecure for production.")
		log.Println("    Set JWT_SECRET environment variable to use a custom secret.")
	}

	// Create Mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(defaultShutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create JWT middleware with static secret
	jwtMiddleware, err := jwt.New(
		jwt.WithSecret([]byte(jwtSecret)),
		jwt.WithExpectedIssuer(jwtIssuer),
	)
	if err != nil {
		log.Fatalf("Failed to create JWT middleware: %v", err)
	}

	// Register modules in correct order
	// 1. JWT middleware (wraps service handlers)
	app.Register(jwtMiddleware)

	// 2. Project module (provides services)
	app.Register(project.NewModule())

	// 3. HTTP module (depends on project module)
	app.Register(http.NewModule(httpPort))

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Print startup information
	printStartupInfo(httpPort, jwtSecret, jwtIssuer)

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

	// Wait for shutdown signal
	os.Exit(<-wait)
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// printStartupInfo displays startup information to the console.
func printStartupInfo(port int, secret, issuer string) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("  JWT Middleware Example - Static Secret Strategy")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("  HTTP Server:   http://localhost:%d", port)
	log.Printf("  Health Check:  http://localhost:%d/health", port)
	log.Printf("  API Endpoint:  http://localhost:%d/api/v1/projects", port)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("  JWT Strategy:  Static HMAC Secret (HS256)")
	log.Printf("  JWT Issuer:    %s", issuer)
	log.Printf("  JWT Secret:    %s", maskSecret(secret))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("  Generate Token:")
	log.Println("    go run cmd/static/generate_token.go")
	log.Println("  ")
	log.Println("  Example Usage:")
	log.Println("    export TOKEN=$(go run cmd/static/generate_token.go | grep '^eyJ' | tr -d '\\n')")
	log.Println("    curl -H \"Authorization: Bearer $TOKEN\" http://localhost:3000/api/v1/projects")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("  Press Ctrl+C to shut down")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// maskSecret masks a secret for display purposes.
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "********"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
