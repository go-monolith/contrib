//go:build tokengen
// +build tokengen

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

func main() {
	// Get user information
	userID := getEnv("USER_ID", "user-123")
	email := getEnv("USER_EMAIL", userID+"@example.com")

	// Start mock JWKS server to get keys
	mockServer, err := NewMockJWKSServer(9000)
	if err != nil {
		log.Fatalf("Failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		log.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Stop(context.Background())

	// Create JWT claims
	claims := jwtgo.MapClaims{
		"iss":   mockServer.Issuer(),
		"aud":   mockServer.Audience(),
		"sub":   userID,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	// Create token with RS256 algorithm
	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)

	// Add kid header
	token.Header["kid"] = mockServer.KeyID()

	// Sign with RSA private key
	tokenString, err := token.SignedString(mockServer.PrivateKey())
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	// Print token information
	printTokenInfo(mockServer, userID, email, tokenString)
}

// printTokenInfo prints the generated token and usage information.
func printTokenInfo(server *MockJWKSServer, userID, email, token string) {
	fmt.Println("JWT Token Generated (RS256)")
	fmt.Println("============================")
	fmt.Printf("Issuer:   %s\n", server.Issuer())
	fmt.Printf("Audience: %s\n", server.Audience())
	fmt.Printf("User ID:  %s\n", userID)
	fmt.Printf("Email:    %s\n", email)
	fmt.Printf("Key ID:   %s\n", server.KeyID())
	fmt.Printf("Expires:  %s\n", time.Now().Add(1*time.Hour).Format(time.RFC3339))
	fmt.Println()
	fmt.Printf("Token: %s\n", token)
	fmt.Println()
	fmt.Println("JWKS Information:")
	fmt.Println("------------------")
	fmt.Printf("JWKS URL: %s\n", server.JWKSURL())
	fmt.Printf("Note: The mock server needs to be running for token validation\n")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("------")
	fmt.Println("Export as environment variable:")
	fmt.Printf("  export JWT_TOKEN=\"%s\"\n", token)
	fmt.Println()
	fmt.Println("Test with curl:")
	fmt.Println("  # List projects")
	fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" http://localhost:3002/api/v1/projects")
	fmt.Println()
	fmt.Println("  # Create project")
	fmt.Println("  curl -X POST http://localhost:3002/api/v1/projects \\")
	fmt.Println("    -H \"Authorization: Bearer $JWT_TOKEN\" \\")
	fmt.Println("    -H \"Content-Type: application/json\" \\")
	fmt.Println("    -d '{\"name\":\"My JWKS Project\",\"description\":\"Test project with RS256\"}'")
	fmt.Println()
	fmt.Println("Generate token for different user:")
	fmt.Println("  USER_ID=user-456 go run -tags=tokengen cmd/jwks/generate_token.go")
}
