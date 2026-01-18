package jwt

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestWithClaimsAndClaimsFromContext(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"role":  "admin",
	}

	// Add claims to context
	ctx = WithClaims(ctx, claims)

	// Retrieve claims from context
	retrievedClaims, ok := ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("ClaimsFromContext() returned false, expected true")
	}

	// Verify claims match
	if retrievedClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", retrievedClaims["sub"])
	}
	if retrievedClaims["email"] != "user@example.com" {
		t.Errorf("Expected email='user@example.com', got: %v", retrievedClaims["email"])
	}
	if retrievedClaims["role"] != "admin" {
		t.Errorf("Expected role='admin', got: %v", retrievedClaims["role"])
	}
}

func TestClaimsFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	claims, ok := ClaimsFromContext(ctx)
	if ok {
		t.Error("ClaimsFromContext() returned true, expected false")
	}
	if claims != nil {
		t.Errorf("ClaimsFromContext() returned non-nil claims: %v", claims)
	}
}

func TestMustClaimsFromContext_Success(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"sub": "user123",
	}

	ctx = WithClaims(ctx, claims)

	retrievedClaims := MustClaimsFromContext(ctx)
	if retrievedClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", retrievedClaims["sub"])
	}
}

func TestMustClaimsFromContext_Panic(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustClaimsFromContext() did not panic")
		} else {
			expectedMsg := "jwt: claims not found in context"
			if r != expectedMsg {
				t.Errorf("Expected panic message '%s', got: %v", expectedMsg, r)
			}
		}
	}()

	// This should panic
	MustClaimsFromContext(ctx)
}

func TestSubjectFromContext_Success(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
	}

	ctx = WithClaims(ctx, claims)

	sub, ok := SubjectFromContext(ctx)
	if !ok {
		t.Fatal("SubjectFromContext() returned false, expected true")
	}
	if sub != "user123" {
		t.Errorf("Expected sub='user123', got: %v", sub)
	}
}

func TestSubjectFromContext_NoClaimsInContext(t *testing.T) {
	ctx := context.Background()

	sub, ok := SubjectFromContext(ctx)
	if ok {
		t.Error("SubjectFromContext() returned true, expected false")
	}
	if sub != "" {
		t.Errorf("SubjectFromContext() returned non-empty string: %v", sub)
	}
}

func TestSubjectFromContext_MissingSubClaim(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"email": "user@example.com",
	}

	ctx = WithClaims(ctx, claims)

	sub, ok := SubjectFromContext(ctx)
	if ok {
		t.Error("SubjectFromContext() returned true when 'sub' claim is missing")
	}
	if sub != "" {
		t.Errorf("SubjectFromContext() returned non-empty string: %v", sub)
	}
}

func TestSubjectFromContext_SubClaimNotString(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"sub": 12345, // Integer instead of string
	}

	ctx = WithClaims(ctx, claims)

	sub, ok := SubjectFromContext(ctx)
	if ok {
		t.Error("SubjectFromContext() returned true when 'sub' claim is not a string")
	}
	if sub != "" {
		t.Errorf("SubjectFromContext() returned non-empty string: %v", sub)
	}
}

func TestIssuerFromContext_Success(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"iss": "https://auth.example.com",
		"sub": "user123",
	}

	ctx = WithClaims(ctx, claims)

	iss, ok := IssuerFromContext(ctx)
	if !ok {
		t.Fatal("IssuerFromContext() returned false, expected true")
	}
	if iss != "https://auth.example.com" {
		t.Errorf("Expected iss='https://auth.example.com', got: %v", iss)
	}
}

func TestIssuerFromContext_NoClaimsInContext(t *testing.T) {
	ctx := context.Background()

	iss, ok := IssuerFromContext(ctx)
	if ok {
		t.Error("IssuerFromContext() returned true, expected false")
	}
	if iss != "" {
		t.Errorf("IssuerFromContext() returned non-empty string: %v", iss)
	}
}

func TestIssuerFromContext_MissingIssClaim(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"sub": "user123",
	}

	ctx = WithClaims(ctx, claims)

	iss, ok := IssuerFromContext(ctx)
	if ok {
		t.Error("IssuerFromContext() returned true when 'iss' claim is missing")
	}
	if iss != "" {
		t.Errorf("IssuerFromContext() returned non-empty string: %v", iss)
	}
}

func TestIssuerFromContext_IssClaimNotString(t *testing.T) {
	ctx := context.Background()
	claims := jwt.MapClaims{
		"iss": 12345, // Integer instead of string
	}

	ctx = WithClaims(ctx, claims)

	iss, ok := IssuerFromContext(ctx)
	if ok {
		t.Error("IssuerFromContext() returned true when 'iss' claim is not a string")
	}
	if iss != "" {
		t.Errorf("IssuerFromContext() returned non-empty string: %v", iss)
	}
}
