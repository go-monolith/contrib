package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateHMACTestKey generates a random HMAC secret key for testing.
//
// Returns a 32-byte random secret suitable for HS256/HS384/HS512.
func GenerateHMACTestKey() []byte {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		panic("failed to generate random secret: " + err.Error())
	}
	return secret
}

// GenerateRSATestKeyPair generates an RSA key pair for testing.
//
// Returns a 2048-bit RSA private key and its corresponding public key.
func GenerateRSATestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate RSA key: " + err.Error())
	}
	return privateKey, &privateKey.PublicKey
}

// GenerateECDSATestKeyPair generates an ECDSA key pair for testing.
//
// Returns a P-256 ECDSA private key and its corresponding public key.
func GenerateECDSATestKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("failed to generate ECDSA key: " + err.Error())
	}
	return privateKey, &privateKey.PublicKey
}

// GenerateValidJWT generates a valid JWT token with custom claims.
//
// Parameters:
//   - key: The signing key ([]byte for HMAC, *rsa.PrivateKey for RSA, *ecdsa.PrivateKey for ECDSA)
//   - claims: Custom claims to include in the token
//
// Returns a signed JWT token string.
//
// Example:
//
//	secret := GenerateHMACTestKey()
//	claims := map[string]interface{}{
//	    "sub": "user123",
//	    "exp": time.Now().Add(1 * time.Hour).Unix(),
//	}
//	token := GenerateValidJWT(secret, claims)
func GenerateValidJWT(key interface{}, claims map[string]interface{}) string {
	// Convert map to jwt.MapClaims
	jwtClaims := jwt.MapClaims{}
	for k, v := range claims {
		jwtClaims[k] = v
	}

	// Determine the signing method based on the key type
	var method jwt.SigningMethod
	switch key.(type) {
	case []byte:
		method = jwt.SigningMethodHS256
	case *rsa.PrivateKey:
		method = jwt.SigningMethodRS256
	case *ecdsa.PrivateKey:
		method = jwt.SigningMethodES256
	default:
		panic("unsupported key type")
	}

	// Create and sign the token
	token := jwt.NewWithClaims(method, jwtClaims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		panic("failed to sign token: " + err.Error())
	}

	return tokenString
}

// GenerateExpiredJWT generates an expired JWT token for testing.
//
// The token is set to expire 1 hour ago.
//
// Parameters:
//   - key: The signing key ([]byte for HMAC, *rsa.PrivateKey for RSA, *ecdsa.PrivateKey for ECDSA)
//
// Returns a signed JWT token string that is expired.
func GenerateExpiredJWT(key interface{}) string {
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	return GenerateValidJWT(key, claims)
}

// GenerateNotYetValidJWT generates a JWT token that is not yet valid for testing.
//
// The token is set to be valid 1 hour from now (nbf claim).
//
// Parameters:
//   - key: The signing key ([]byte for HMAC, *rsa.PrivateKey for RSA, *ecdsa.PrivateKey for ECDSA)
//
// Returns a signed JWT token string that is not yet valid.
func GenerateNotYetValidJWT(key interface{}) string {
	claims := map[string]interface{}{
		"sub": "user123",
		"nbf": time.Now().Add(1 * time.Hour).Unix(),
		"exp": time.Now().Add(2 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	return GenerateValidJWT(key, claims)
}

// GenerateInvalidSignatureJWT generates a JWT token with an invalid signature.
//
// This creates a token signed with one key, but returns a token where the
// signature has been tampered with.
//
// Returns a JWT token string with an invalid signature.
func GenerateInvalidSignatureJWT() string {
	// Generate a valid token
	secret := GenerateHMACTestKey()
	claims := map[string]interface{}{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	validToken := GenerateValidJWT(secret, claims)

	// Tamper with the signature by changing the last character
	if len(validToken) > 0 {
		tokenBytes := []byte(validToken)
		if tokenBytes[len(tokenBytes)-1] == 'A' {
			tokenBytes[len(tokenBytes)-1] = 'B'
		} else {
			tokenBytes[len(tokenBytes)-1] = 'A'
		}
		return string(tokenBytes)
	}

	return validToken
}

// GenerateTokenWithIssuer generates a JWT token with a specific issuer claim.
//
// Parameters:
//   - key: The signing key
//   - issuer: The issuer value to include in the "iss" claim
//
// Returns a signed JWT token string with the issuer claim.
func GenerateTokenWithIssuer(key interface{}, issuer string) string {
	claims := map[string]interface{}{
		"sub": "user123",
		"iss": issuer,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	return GenerateValidJWT(key, claims)
}

// GenerateTokenWithAudience generates a JWT token with a specific audience claim.
//
// Parameters:
//   - key: The signing key
//   - audience: The audience value(s) to include in the "aud" claim
//
// Returns a signed JWT token string with the audience claim.
func GenerateTokenWithAudience(key interface{}, audience interface{}) string {
	claims := map[string]interface{}{
		"sub": "user123",
		"aud": audience,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	return GenerateValidJWT(key, claims)
}

// GenerateTokenWithCustomClaims generates a JWT token with completely custom claims.
//
// This is useful when you need full control over the token's claims.
//
// Parameters:
//   - key: The signing key
//   - customClaims: All claims to include in the token
//
// Returns a signed JWT token string.
func GenerateTokenWithCustomClaims(key interface{}, customClaims map[string]interface{}) string {
	return GenerateValidJWT(key, customClaims)
}

// GenerateTokenWithAlgorithm generates a JWT token with a specific algorithm.
//
// Parameters:
//   - key: The signing key
//   - method: The JWT signing method (e.g., jwt.SigningMethodHS256, jwt.SigningMethodRS512)
//   - claims: Custom claims to include
//
// Returns a signed JWT token string.
func GenerateTokenWithAlgorithm(key interface{}, method jwt.SigningMethod, claims map[string]interface{}) string {
	jwtClaims := jwt.MapClaims{}
	for k, v := range claims {
		jwtClaims[k] = v
	}

	token := jwt.NewWithClaims(method, jwtClaims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		panic("failed to sign token: " + err.Error())
	}

	return tokenString
}

// GenerateTokenWithKid generates a JWT token with a specific key ID (kid) in the header.
//
// Parameters:
//   - key: The signing key
//   - kid: The key ID to include in the token header
//   - claims: Custom claims to include
//
// Returns a signed JWT token string with the kid header.
func GenerateTokenWithKid(key interface{}, kid string, claims map[string]interface{}) string {
	jwtClaims := jwt.MapClaims{}
	for k, v := range claims {
		jwtClaims[k] = v
	}

	// Determine the signing method
	var method jwt.SigningMethod
	switch key.(type) {
	case []byte:
		method = jwt.SigningMethodHS256
	case *rsa.PrivateKey:
		method = jwt.SigningMethodRS256
	case *ecdsa.PrivateKey:
		method = jwt.SigningMethodES256
	default:
		panic("unsupported key type")
	}

	token := jwt.NewWithClaims(method, jwtClaims)
	token.Header["kid"] = kid

	tokenString, err := token.SignedString(key)
	if err != nil {
		panic("failed to sign token: " + err.Error())
	}

	return tokenString
}

// MockJWKSKey represents a public key to be served by a mock JWKS server.
type MockJWKSKey struct {
	Kid       string
	PublicKey interface{} // *rsa.PublicKey or *ecdsa.PublicKey
}

// CreateMockJWKSServer creates a mock HTTP server that serves a JWKS endpoint.
//
// The server returns a JSON Web Key Set (JWKS) containing the provided public keys.
// This is useful for testing JWKS-based JWT validation.
//
// Parameters:
//   - keys: Slice of MockJWKSKey containing the keys to serve
//
// Returns an *httptest.Server that serves the JWKS at the root path.
// The caller should call server.Close() when done.
//
// Example:
//
//	_, publicKey := GenerateRSATestKeyPair()
//	server := CreateMockJWKSServer([]*MockJWKSKey{
//	    {Kid: "key1", PublicKey: publicKey},
//	})
//	defer server.Close()
func CreateMockJWKSServer(keys []*MockJWKSKey) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": buildJWKSKeys(keys),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
}

// buildJWKSKeys converts MockJWKSKey slice to JWKS JSON format.
func buildJWKSKeys(keys []*MockJWKSKey) []map[string]interface{} {
	var jwkKeys []map[string]interface{}

	for _, key := range keys {
		var jwk map[string]interface{}

		switch pubKey := key.PublicKey.(type) {
		case *rsa.PublicKey:
			jwk = map[string]interface{}{
				"kty": "RSA",
				"kid": key.Kid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes()),
			}

		case *ecdsa.PublicKey:
			var crv string
			switch pubKey.Curve {
			case elliptic.P256():
				crv = "P-256"
			case elliptic.P384():
				crv = "P-384"
			case elliptic.P521():
				crv = "P-521"
			default:
				panic(fmt.Sprintf("unsupported ECDSA curve: %v", pubKey.Curve))
			}

			jwk = map[string]interface{}{
				"kty": "EC",
				"kid": key.Kid,
				"use": "sig",
				"alg": "ES256",
				"crv": crv,
				"x":   base64.RawURLEncoding.EncodeToString(pubKey.X.Bytes()),
				"y":   base64.RawURLEncoding.EncodeToString(pubKey.Y.Bytes()),
			}

		default:
			panic(fmt.Sprintf("unsupported key type: %T", key.PublicKey))
		}

		jwkKeys = append(jwkKeys, jwk)
	}

	return jwkKeys
}
