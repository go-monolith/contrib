//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultJWTSecret = "dev-secret-change-in-production"
	defaultJWTIssuer = "jwt-example-static"
)

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Load configuration from environment
	secret := getEnv("JWT_SECRET", defaultJWTSecret)
	issuer := getEnv("JWT_ISSUER", defaultJWTIssuer)
	userID := getEnv("USER_ID", "user-123")
	email := getEnv("USER_EMAIL", userID+"@example.com")

	// Create JWT claims
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   issuer,                       // Issuer
		"sub":   userID,                       // Subject (user ID)
		"email": email,                        // Email (custom claim)
		"iat":   now.Unix(),                   // Issued at
		"exp":   now.Add(1 * time.Hour).Unix(), // Expires in 1 hour
	}

	// Create token with HS256 algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}

	// Print token information
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  JWT Token Generated (Static Secret Strategy)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  User ID:    %s\n", userID)
	fmt.Printf("  Email:      %s\n", email)
	fmt.Printf("  Issuer:     %s\n", issuer)
	fmt.Printf("  Expires:    %s\n", time.Unix(claims["exp"].(int64), 0).Format(time.RFC3339))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Token:")
	fmt.Println()
	fmt.Println(tokenString)
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Usage:")
	fmt.Println()
	fmt.Println("  # Export as environment variable")
	fmt.Println("  export JWT_TOKEN=\"" + tokenString + "\"")
	fmt.Println()
	fmt.Println("  # Use with curl")
	fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" \\")
	fmt.Println("    http://localhost:3000/api/v1/projects")
	fmt.Println()
	fmt.Println("  # Create a project")
	fmt.Println("  curl -X POST -H \"Authorization: Bearer $JWT_TOKEN\" \\")
	fmt.Println("    -H \"Content-Type: application/json\" \\")
	fmt.Println("    -d '{\"name\":\"My Project\",\"description\":\"Test project\"}' \\")
	fmt.Println("    http://localhost:3000/api/v1/projects")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("  Customize with environment variables:")
	fmt.Println("    USER_ID=user-456 USER_EMAIL=alice@example.com go run cmd/static/generate_token.go")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
