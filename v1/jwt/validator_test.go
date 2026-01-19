package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
	"github.com/golang-jwt/jwt/v5"
)

// Mock KeyProvider for testing
type mockKeyProvider struct {
	key interface{}
	err error
}

func (m *mockKeyProvider) GetKey(ctx context.Context, kid string) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.key, nil
}

// Helper function to generate a valid HMAC JWT
func generateHMACToken(secret []byte, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// Helper function to generate a valid RSA JWT
func generateRSAToken(privateKey *rsa.PrivateKey, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// Helper function to generate a valid ECDSA JWT
func generateECDSAToken(privateKey *ecdsa.PrivateKey, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(privateKey)
}

func TestTokenValidator_ValidToken_HMAC(t *testing.T) {
	secret := []byte("test-secret-key")
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString, err := generateHMACToken(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	provider := &mockKeyProvider{key: secret}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	parsedClaims, err := validator.Validate(ctx, tokenString)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if parsedClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", parsedClaims["sub"])
	}
}

func TestTokenValidator_ValidToken_RSA(t *testing.T) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	claims := jwt.MapClaims{
		"sub": "user456",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString, err := generateRSAToken(privateKey, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	provider := &mockKeyProvider{key: publicKey}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	parsedClaims, err := validator.Validate(ctx, tokenString)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if parsedClaims["sub"] != "user456" {
		t.Errorf("Expected sub='user456', got: %v", parsedClaims["sub"])
	}
}

func TestTokenValidator_ValidToken_ECDSA(t *testing.T) {
	// Generate ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	claims := jwt.MapClaims{
		"sub": "user789",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString, err := generateECDSAToken(privateKey, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	provider := &mockKeyProvider{key: publicKey}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	parsedClaims, err := validator.Validate(ctx, tokenString)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if parsedClaims["sub"] != "user789" {
		t.Errorf("Expected sub='user789', got: %v", parsedClaims["sub"])
	}
}

func TestTokenValidator_InvalidSignature(t *testing.T) {
	secret := []byte("correct-secret")
	wrongSecret := []byte("wrong-secret")

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	// Generate token with one secret
	tokenString, err := generateHMACToken(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with a different secret
	provider := &mockKeyProvider{key: wrongSecret}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	_, err = validator.Validate(ctx, tokenString)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got: %v", err)
	}
}

func TestTokenValidator_UnsupportedAlgorithm(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	// Generate token with HS256
	tokenString, err := generateHMACToken(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Configure validator to only allow HS512 (not HS256)
	provider := &mockKeyProvider{key: secret}
	config := &Config{
		AllowedAlgorithms: []string{"HS512"},
	}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	_, err = validator.Validate(ctx, tokenString)
	if err != ErrUnsupportedAlgorithm {
		t.Errorf("Expected ErrUnsupportedAlgorithm, got: %v", err)
	}
}

func TestTokenValidator_MalformedToken(t *testing.T) {
	testCases := []struct {
		name        string
		tokenString string
	}{
		{
			name:        "empty string",
			tokenString: "",
		},
		{
			name:        "random string",
			tokenString: "not.a.jwt.token",
		},
		{
			name:        "invalid base64",
			tokenString: "invalid!!!.base64###.data$$$",
		},
		{
			name:        "missing signature",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0",
		},
	}

	secret := []byte("test-secret")
	provider := &mockKeyProvider{key: secret}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.Validate(ctx, tc.tokenString)
			if err == nil {
				t.Error("Validate() should fail for malformed token")
			}
		})
	}
}

func TestTokenValidator_KeyProviderError(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString, err := generateHMACToken(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Provider that returns an error
	providerErr := errors.New("key provider error")
	provider := &mockKeyProvider{err: providerErr}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	_, err = validator.Validate(ctx, tokenString)
	if err == nil {
		t.Error("Validate() should fail when key provider returns error")
	}
}

func TestTokenValidator_TokenWithKid(t *testing.T) {
	secret := []byte("test-secret")

	// Create token with kid in header
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	token.Header["kid"] = "key-id-123"

	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	provider := &mockKeyProvider{key: secret}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	parsedClaims, err := validator.Validate(ctx, tokenString)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if parsedClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", parsedClaims["sub"])
	}
}

func TestTokenValidator_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret")

	// Create token that expired 1 hour ago
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}

	tokenString, err := generateHMACToken(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	provider := &mockKeyProvider{key: secret}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	ctx := context.Background()
	_, err = validator.Validate(ctx, tokenString)

	// The jwt library may return ErrTokenExpired or we may catch it
	if err == nil {
		t.Error("Validate() should fail for expired token")
	}
}

func TestNewTokenValidator(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{}

	validator := NewTokenValidator(provider, config, nil)
	if validator == nil {
		t.Fatal("NewTokenValidator() returned nil")
	}
	if validator.keyProvider != provider {
		t.Error("Validator keyProvider not set correctly")
	}
	if validator.config != config {
		t.Error("Validator config not set correctly")
	}
}
// Tests for validateStandardClaims

func TestValidateStandardClaims_ExpiredToken(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token expired 2 minutes ago (beyond clock skew)
	claims := jwt.MapClaims{
		"exp": time.Now().Add(-2 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
}

func TestValidateStandardClaims_ExpiredToken_WithinClockSkew(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 2 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token expired 1 minute ago (within clock skew of 2 minutes)
	claims := jwt.MapClaims{
		"exp": time.Now().Add(-1 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != nil {
		t.Errorf("Should accept token within clock skew, got error: %v", err)
	}
}

func TestValidateStandardClaims_NotYetValid(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token valid 2 minutes from now (beyond clock skew)
	claims := jwt.MapClaims{
		"nbf": time.Now().Add(2 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != ErrTokenNotYetValid {
		t.Errorf("Expected ErrTokenNotYetValid, got: %v", err)
	}
}

func TestValidateStandardClaims_NotYetValid_WithinClockSkew(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 2 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token valid 1 minute from now (within clock skew)
	claims := jwt.MapClaims{
		"nbf": time.Now().Add(1 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != nil {
		t.Errorf("Should accept token within clock skew, got error: %v", err)
	}
}

func TestValidateStandardClaims_InvalidIssuedAt(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token issued 2 minutes in the future (beyond clock skew)
	claims := jwt.MapClaims{
		"iat": time.Now().Add(2 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != ErrInvalidIssuedAt {
		t.Errorf("Expected ErrInvalidIssuedAt, got: %v", err)
	}
}

func TestValidateStandardClaims_ValidIssuedAt_WithinClockSkew(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ClockSkew: 2 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token issued 1 minute in the future (within clock skew)
	claims := jwt.MapClaims{
		"iat": time.Now().Add(1 * time.Minute).Unix(),
	}

	err := validator.validateStandardClaims(claims)
	if err != nil {
		t.Errorf("Should accept token within clock skew, got error: %v", err)
	}
}

// Tests for validateIssuer

func TestValidateIssuer_ValidIssuer(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedIssuer: "https://auth.example.com",
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"iss": "https://auth.example.com",
	}

	err := validator.validateIssuer(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateIssuer_InvalidIssuer(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedIssuer: "https://auth.example.com",
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"iss": "https://different-issuer.com",
	}

	err := validator.validateIssuer(claims)
	if err != ErrInvalidIssuer {
		t.Errorf("Expected ErrInvalidIssuer, got: %v", err)
	}
}

func TestValidateIssuer_MissingIssuer(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedIssuer: "https://auth.example.com",
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{}

	err := validator.validateIssuer(claims)
	if err != ErrInvalidIssuer {
		t.Errorf("Expected ErrInvalidIssuer, got: %v", err)
	}
}

func TestValidateIssuer_NoExpectedIssuer(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"iss": "https://any-issuer.com",
	}

	err := validator.validateIssuer(claims)
	if err != nil {
		t.Errorf("Should skip validation when no expected issuer, got error: %v", err)
	}
}

// Tests for validateAudience

func TestValidateAudience_SingleAudience_Match(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedAudience: []string{"https://api.example.com"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"aud": "https://api.example.com",
	}

	err := validator.validateAudience(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateAudience_MultipleAudiences_Match(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedAudience: []string{"https://api1.example.com", "https://api2.example.com"},
	}
	validator := NewTokenValidator(provider, config, nil)

	// Token has one of the expected audiences
	claims := jwt.MapClaims{
		"aud": []interface{}{"https://api2.example.com", "https://other.com"},
	}

	err := validator.validateAudience(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateAudience_NoMatch(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedAudience: []string{"https://api.example.com"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"aud": "https://different-api.com",
	}

	err := validator.validateAudience(claims)
	if err != ErrInvalidAudience {
		t.Errorf("Expected ErrInvalidAudience, got: %v", err)
	}
}

func TestValidateAudience_MissingAudience(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		ExpectedAudience: []string{"https://api.example.com"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{}

	err := validator.validateAudience(claims)
	if err != ErrInvalidAudience {
		t.Errorf("Expected ErrInvalidAudience, got: %v", err)
	}
}

func TestValidateAudience_NoExpectedAudience(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"aud": "https://any-audience.com",
	}

	err := validator.validateAudience(claims)
	if err != nil {
		t.Errorf("Should skip validation when no expected audience, got error: %v", err)
	}
}

// Tests for validateRequiredClaims

func TestValidateRequiredClaims_AllPresent(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		RequiredClaims: []string{"sub", "email", "role"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"role":  "admin",
	}

	err := validator.validateRequiredClaims(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateRequiredClaims_MissingClaim(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		RequiredClaims: []string{"sub", "email", "role"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		// "role" is missing
	}

	err := validator.validateRequiredClaims(claims)
	if err != ErrMissingRequiredClaim {
		t.Errorf("Expected ErrMissingRequiredClaim, got: %v", err)
	}
}

func TestValidateRequiredClaims_EmptyString(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		RequiredClaims: []string{"sub", "email"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "", // Empty string
	}

	err := validator.validateRequiredClaims(claims)
	if err != ErrMissingRequiredClaim {
		t.Errorf("Expected ErrMissingRequiredClaim for empty string, got: %v", err)
	}
}

func TestValidateRequiredClaims_NilValue(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{
		RequiredClaims: []string{"sub"},
	}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"sub": nil,
	}

	err := validator.validateRequiredClaims(claims)
	if err != ErrMissingRequiredClaim {
		t.Errorf("Expected ErrMissingRequiredClaim for nil value, got: %v", err)
	}
}

func TestValidateRequiredClaims_NoRequiredClaims(t *testing.T) {
	provider := &mockKeyProvider{key: []byte("test")}
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	claims := jwt.MapClaims{
		"sub": "user123",
	}

	err := validator.validateRequiredClaims(claims)
	if err != nil {
		t.Errorf("Should skip validation when no required claims, got error: %v", err)
	}
}

// Test refresh-on-signature-failure strategy

// TestTokenValidator_RefreshOnSignatureFailure tests automatic JWKS refresh on signature error.
func TestTokenValidator_RefreshOnSignatureFailure(t *testing.T) {
	// Generate two different RSA key pairs
	_, oldPublicKey := testutil.GenerateRSATestKeyPair()
	newPrivateKey, newPublicKey := testutil.GenerateRSATestKeyPair()

	// Track which key set is returned
	useNewKey := false
	var keyMu sync.Mutex

	// Create mock JWKS server that simulates key rotation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyMu.Lock()
		defer keyMu.Unlock()

		var publicKey *rsa.PublicKey
		if useNewKey {
			publicKey = newPublicKey
		} else {
			publicKey = oldPublicKey
		}

		rsaE := publicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "key1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	// Create JWKS provider and validator
	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)
	config := &Config{
		ClockSkew: 1 * time.Minute,
	}
	validator := NewTokenValidator(provider, config, nil)

	// Initial cache refresh with old key
	ctx := context.Background()
	if err := provider.RefreshCache(ctx); err != nil {
		t.Fatalf("Initial refresh failed: %v", err)
	}

	// Generate token with NEW key (simulating key rotation)
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	tokenString := testutil.GenerateTokenWithKid(newPrivateKey, "key1", claims)

	// First validation attempt should fail (old key in cache)
	// But validator should auto-refresh and succeed

	// Switch server to return new key on next refresh
	keyMu.Lock()
	useNewKey = true
	keyMu.Unlock()

	// Validate - should trigger refresh and succeed
	parsedClaims, err := validator.Validate(ctx, tokenString)
	if err != nil {
		t.Fatalf("Validation should succeed after refresh, got error: %v", err)
	}

	if parsedClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", parsedClaims["sub"])
	}
}

// TestTokenValidator_RefreshOnSignatureFailure_StillFails tests when token is truly invalid.
func TestTokenValidator_RefreshOnSignatureFailure_StillFails(t *testing.T) {
	_, rsaPublicKey := testutil.GenerateRSATestKeyPair()

	// Create mock JWKS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsaE := rsaPublicKey.E
		rsaN := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())
		eBytes := make([]byte, 4)
		eBytes[0] = byte(rsaE >> 24)
		eBytes[1] = byte(rsaE >> 16)
		eBytes[2] = byte(rsaE >> 8)
		eBytes[3] = byte(rsaE)
		i := 0
		for i < len(eBytes) && eBytes[i] == 0 {
			i++
		}
		eBytes = eBytes[i:]
		rsaEEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := fmt.Sprintf(`{
			"keys": [
				{
					"kty": "RSA",
					"kid": "key1",
					"use": "sig",
					"n": "%s",
					"e": "%s"
				}
			]
		}`, rsaN, rsaEEncoded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	}))
	defer server.Close()

	cache := NewJWKSCache(15 * time.Minute)
	provider := NewJWKSKeyProvider(server.URL, cache, 10*time.Second)
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	// Create token with invalid signature
	tokenString := testutil.GenerateInvalidSignatureJWT()

	// Validation should fail even after refresh
	ctx := context.Background()
	_, err := validator.Validate(ctx, tokenString)
	if err == nil {
		t.Error("Expected validation to fail for invalid signature")
	}

	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got: %v", err)
	}
}

// TestTokenValidator_NoRefreshWithStaticProvider tests that refresh doesn't happen with static keys.
func TestTokenValidator_NoRefreshWithStaticProvider(t *testing.T) {
	secret := testutil.GenerateHMACTestKey()
	provider := NewStaticKeyProvider(secret)
	config := &Config{}
	validator := NewTokenValidator(provider, config, nil)

	// Create token with invalid signature
	tokenString := testutil.GenerateInvalidSignatureJWT()

	// Validation should fail immediately without refresh attempt
	ctx := context.Background()
	_, err := validator.Validate(ctx, tokenString)
	if err == nil {
		t.Error("Expected validation to fail")
	}

	// Should get invalid signature error (not a fetch error)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got: %v", err)
	}
}
