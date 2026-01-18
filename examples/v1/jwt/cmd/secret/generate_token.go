//go:build tokengen
// +build tokengen

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

func main() {
	// Get tenant name (issuer)
	tenant := getEnv("TENANT", "tenant-1")

	// Get user information
	userID := getEnv("USER_ID", "user-123")
	email := getEnv("USER_EMAIL", userID+"@example.com")

	// Lookup secret for tenant from environment
	secret := getSecretForTenant(tenant)
	if secret == "" {
		log.Fatalf("No secret configured for tenant '%s'. Please set the corresponding TENANT_N_SECRET environment variable.", tenant)
	}

	// Create JWT claims
	claims := jwtgo.MapClaims{
		"iss":   tenant,
		"sub":   userID,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	// Create and sign token
	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	// Print token information
	printTokenInfo(tenant, userID, email, tokenString)
}

// getSecretForTenant retrieves the secret for a given tenant from environment variables.
// Searches for TENANT_N_NAME matching the tenant name and returns TENANT_N_SECRET.
func getSecretForTenant(tenant string) string {
	// Try to find tenant in environment variables
	for i := 1; i <= 10; i++ {
		nameKey := fmt.Sprintf("TENANT_%d_NAME", i)
		secretKey := fmt.Sprintf("TENANT_%d_SECRET", i)

		name := os.Getenv(nameKey)
		secret := os.Getenv(secretKey)

		if name == tenant {
			return secret
		}
	}

	return ""
}

// printTokenInfo prints the generated token and usage information.
func printTokenInfo(tenant, userID, email, token string) {
	fmt.Println("JWT Token Generated")
	fmt.Println("===================")
	fmt.Printf("Tenant:  %s\n", tenant)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Email:   %s\n", email)
	fmt.Printf("Expires: %s\n", time.Now().Add(1*time.Hour).Format(time.RFC3339))
	fmt.Println()
	fmt.Printf("Token: %s\n", token)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("------")
	fmt.Println("Export as environment variable:")
	fmt.Printf("  export JWT_TOKEN=\"%s\"\n", token)
	fmt.Println()
	fmt.Println("Test with curl:")
	fmt.Println("  # List projects")
	fmt.Println("  curl -H \"Authorization: Bearer $JWT_TOKEN\" http://localhost:3001/api/v1/projects")
	fmt.Println()
	fmt.Println("  # Create project")
	fmt.Println("  curl -X POST http://localhost:3001/api/v1/projects \\")
	fmt.Println("    -H \"Authorization: Bearer $JWT_TOKEN\" \\")
	fmt.Println("    -H \"Content-Type: application/json\" \\")
	fmt.Printf("    -d '{\"name\":\"My Project for %s\",\"description\":\"Test project\"}'\n", tenant)
	fmt.Println()
	fmt.Println("Generate token for different tenant:")
	fmt.Printf("  TENANT=tenant-2 USER_ID=user-456 go run cmd/secret/generate_token.go\n")
}
