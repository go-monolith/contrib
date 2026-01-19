package testutil

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateHMACTestKey(t *testing.T) {
	key := GenerateHMACTestKey()

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	// Generate another key and ensure they're different
	key2 := GenerateHMACTestKey()
	if string(key) == string(key2) {
		t.Error("Generated keys should be different")
	}
}

func TestGenerateRSATestKeyPair(t *testing.T) {
	privateKey, publicKey := GenerateRSATestKeyPair()

	if privateKey == nil {
		t.Fatal("Private key is nil")
	}
	if publicKey == nil {
		t.Fatal("Public key is nil")
	}

	// Verify the public key matches the private key
	if privateKey.N.Cmp(publicKey.N) != 0 {
		t.Error("Public key doesn't match private key")
	}
}

func TestGenerateECDSATestKeyPair(t *testing.T) {
	privateKey, publicKey := GenerateECDSATestKeyPair()

	if privateKey == nil {
		t.Fatal("Private key is nil")
	}
	if publicKey == nil {
		t.Fatal("Public key is nil")
	}

	// Verify the public key matches the private key
	if privateKey.X.Cmp(publicKey.X) != 0 || privateKey.Y.Cmp(publicKey.Y) != 0 {
		t.Error("Public key doesn't match private key")
	}
}

func TestGenerateValidJWT_HMAC(t *testing.T) {
	secret := GenerateHMACTestKey()
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateValidJWT(secret, claims)

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token should be valid")
	}

	tokenClaims := token.Claims.(jwt.MapClaims)
	if tokenClaims["sub"] != "user123" {
		t.Errorf("Expected sub='user123', got: %v", tokenClaims["sub"])
	}
}

func TestGenerateValidJWT_RSA(t *testing.T) {
	privateKey, publicKey := GenerateRSATestKeyPair()
	claims := map[string]interface{}{
		"sub": "user456",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateValidJWT(privateKey, claims)

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token should be valid")
	}

	tokenClaims := token.Claims.(jwt.MapClaims)
	if tokenClaims["sub"] != "user456" {
		t.Errorf("Expected sub='user456', got: %v", tokenClaims["sub"])
	}
}

func TestGenerateValidJWT_ECDSA(t *testing.T) {
	privateKey, publicKey := GenerateECDSATestKeyPair()
	claims := map[string]interface{}{
		"sub": "user789",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateValidJWT(privateKey, claims)

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token should be valid")
	}

	tokenClaims := token.Claims.(jwt.MapClaims)
	if tokenClaims["sub"] != "user789" {
		t.Errorf("Expected sub='user789', got: %v", tokenClaims["sub"])
	}
}

func TestGenerateExpiredJWT(t *testing.T) {
	secret := GenerateHMACTestKey()
	tokenString := GenerateExpiredJWT(secret)

	// Parse the token (should fail validation due to expiration)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	// Token should be parsed but not valid
	if err == nil && token.Valid {
		t.Error("Token should not be valid (expired)")
	}
}

func TestGenerateNotYetValidJWT(t *testing.T) {
	secret := GenerateHMACTestKey()
	tokenString := GenerateNotYetValidJWT(secret)

	// Parse the token (should fail validation due to nbf)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	// Token should be parsed but not valid
	if err == nil && token.Valid {
		t.Error("Token should not be valid (not yet valid)")
	}
}

func TestGenerateInvalidSignatureJWT(t *testing.T) {
	tokenString := GenerateInvalidSignatureJWT()

	// Try to parse with a random secret
	secret := GenerateHMACTestKey()
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	// Should fail due to invalid signature
	if err == nil {
		t.Error("Token with invalid signature should fail to parse")
	}
}

func TestGenerateTokenWithIssuer(t *testing.T) {
	secret := GenerateHMACTestKey()
	issuer := "https://auth.example.com"
	tokenString := GenerateTokenWithIssuer(secret, issuer)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["iss"] != issuer {
		t.Errorf("Expected iss='%s', got: %v", issuer, claims["iss"])
	}
}

func TestGenerateTokenWithAudience_String(t *testing.T) {
	secret := GenerateHMACTestKey()
	audience := "https://api.example.com"
	tokenString := GenerateTokenWithAudience(secret, audience)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["aud"] != audience {
		t.Errorf("Expected aud='%s', got: %v", audience, claims["aud"])
	}
}

func TestGenerateTokenWithAudience_Array(t *testing.T) {
	secret := GenerateHMACTestKey()
	audience := []string{"https://api1.example.com", "https://api2.example.com"}
	tokenString := GenerateTokenWithAudience(secret, audience)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	// The aud claim should be present
	if claims["aud"] == nil {
		t.Error("Expected aud claim to be present")
	}
}

func TestGenerateTokenWithCustomClaims(t *testing.T) {
	secret := GenerateHMACTestKey()
	customClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"role":  "admin",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateTokenWithCustomClaims(secret, customClaims)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["email"] != "user@example.com" {
		t.Errorf("Expected email='user@example.com', got: %v", claims["email"])
	}
	if claims["role"] != "admin" {
		t.Errorf("Expected role='admin', got: %v", claims["role"])
	}
}

func TestGenerateTokenWithAlgorithm(t *testing.T) {
	secret := GenerateHMACTestKey()
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateTokenWithAlgorithm(secret, jwt.SigningMethodHS512, claims)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if token.Method.Alg() != "HS512" {
		t.Errorf("Expected algorithm HS512, got: %s", token.Method.Alg())
	}
}

func TestGenerateTokenWithKid(t *testing.T) {
	secret := GenerateHMACTestKey()
	kid := "key-id-123"
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	tokenString := GenerateTokenWithKid(secret, kid, claims)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if token.Header["kid"] != kid {
		t.Errorf("Expected kid='%s', got: %v", kid, token.Header["kid"])
	}
}

func TestKeyTypeDetection(t *testing.T) {
	// Test that different key types generate tokens with correct algorithms
	testCases := []struct {
		name             string
		keyGen           func() interface{}
		expectedAlg      string
		verificationKey  func(interface{}) interface{}
	}{
		{
			name:        "HMAC",
			keyGen:      func() interface{} { return GenerateHMACTestKey() },
			expectedAlg: "HS256",
			verificationKey: func(key interface{}) interface{} { return key },
		},
		{
			name: "RSA",
			keyGen: func() interface{} {
				priv, _ := GenerateRSATestKeyPair()
				return priv
			},
			expectedAlg: "RS256",
			verificationKey: func(key interface{}) interface{} {
				return &key.(*rsa.PrivateKey).PublicKey
			},
		},
		{
			name: "ECDSA",
			keyGen: func() interface{} {
				priv, _ := GenerateECDSATestKeyPair()
				return priv
			},
			expectedAlg: "ES256",
			verificationKey: func(key interface{}) interface{} {
				return &key.(*ecdsa.PrivateKey).PublicKey
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := tc.keyGen()
			claims := map[string]interface{}{
				"sub": "user123",
				"exp": time.Now().Add(1 * time.Hour).Unix(),
			}

			tokenString := GenerateValidJWT(key, claims)

			verifyKey := tc.verificationKey(key)
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return verifyKey, nil
			})

			if err != nil {
				t.Fatalf("Failed to parse token: %v", err)
			}

			if token.Method.Alg() != tc.expectedAlg {
				t.Errorf("Expected algorithm %s, got: %s", tc.expectedAlg, token.Method.Alg())
			}
		})
	}
}
