package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"` // Key Type
	Use string `json:"use"` // Public Key Use
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm
	N   string `json:"n"`   // Modulus
	E   string `json:"e"`   // Exponent
}

// JWKSet represents a JSON Web Key Set.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// MockJWKSServer provides a mock JWKS endpoint for testing.
type MockJWKSServer struct {
	port       int
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
	issuer     string
	audience   string
	server     *http.Server
	mu         sync.RWMutex
}

// NewMockJWKSServer creates a new mock JWKS server.
func NewMockJWKSServer(port int) (*MockJWKSServer, error) {
	// Generate RSA key pair (2048 bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &MockJWKSServer{
		port:       port,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		keyID:      "mock-key-1",
		issuer:     "mock-jwks-issuer",
		audience:   "mock-jwks-audience",
	}, nil
}

// Start starts the mock JWKS server.
func (m *MockJWKSServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", m.handleJWKS)
	mux.HandleFunc("/health", m.handleHealth)

	m.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", m.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("[jwks] Mock JWKS server starting on port %d", m.port)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[jwks] Server error: %v", err)
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop stops the mock JWKS server.
func (m *MockJWKSServer) Stop(ctx context.Context) error {
	if m.server != nil {
		log.Println("[jwks] Stopping mock JWKS server...")
		return m.server.Shutdown(ctx)
	}
	return nil
}

// handleJWKS handles requests to the JWKS endpoint.
func (m *MockJWKSServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert RSA public key to JWK format
	jwk := m.publicKeyToJWK()

	// Create JWKS response
	jwks := JWKSet{
		Keys: []JWK{jwk},
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(jwks); err != nil {
		log.Printf("[jwks] Error encoding JWKS: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleHealth handles health check requests.
func (m *MockJWKSServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "mock-jwks-server",
	})
}

// publicKeyToJWK converts an RSA public key to JWK format.
func (m *MockJWKSServer) publicKeyToJWK() JWK {
	// Extract modulus and exponent from public key
	nBytes := m.publicKey.N.Bytes()
	eBytes := big.NewInt(int64(m.publicKey.E)).Bytes()

	// Base64url encode (without padding)
	n := base64.RawURLEncoding.EncodeToString(nBytes)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	return JWK{
		Kty: "RSA",
		Use: "sig",
		Kid: m.keyID,
		Alg: "RS256",
		N:   n,
		E:   e,
	}
}

// PrivateKey returns the private key for signing tokens.
func (m *MockJWKSServer) PrivateKey() *rsa.PrivateKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.privateKey
}

// PublicKey returns the public key.
func (m *MockJWKSServer) PublicKey() *rsa.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publicKey
}

// KeyID returns the key ID.
func (m *MockJWKSServer) KeyID() string {
	return m.keyID
}

// Issuer returns the issuer claim value.
func (m *MockJWKSServer) Issuer() string {
	return m.issuer
}

// Audience returns the audience claim value.
func (m *MockJWKSServer) Audience() string {
	return m.audience
}

// JWKSURL returns the JWKS endpoint URL.
func (m *MockJWKSServer) JWKSURL() string {
	return fmt.Sprintf("http://localhost:%d/.well-known/jwks.json", m.port)
}

// PublicKeyPEM returns the public key in PEM format (for debugging).
func (m *MockJWKSServer) PublicKeyPEM() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(m.publicKey)
	if err != nil {
		return "", err
	}

	// Convert to PEM format (without actual encoding, just for display)
	b64 := base64.StdEncoding.EncodeToString(pubKeyBytes)
	return fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----", b64), nil
}
