package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// Helper function to generate RSA PEM in PKIX format (PUBLIC KEY)
func generateRSAPublicKeyPEM_PKIX(t *testing.T) ([]byte, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return pem.EncodeToMemory(pemBlock), &privateKey.PublicKey
}

// Helper function to generate RSA PEM in PKCS#1 format (RSA PUBLIC KEY)
func generateRSAPublicKeyPEM_PKCS1(t *testing.T) ([]byte, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pubKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

	pemBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return pem.EncodeToMemory(pemBlock), &privateKey.PublicKey
}

// Helper function to generate ECDSA PEM in PKIX format (PUBLIC KEY)
func generateECDSAPublicKeyPEM_PKIX(t *testing.T) ([]byte, *ecdsa.PublicKey) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return pem.EncodeToMemory(pemBlock), &privateKey.PublicKey
}

func TestParseRSAPublicKeyFromPEM_PKIX(t *testing.T) {
	pemData, expectedKey := generateRSAPublicKeyPEM_PKIX(t)

	parsedKey, err := ParseRSAPublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromPEM() failed: %v", err)
	}

	// Compare key parameters
	if parsedKey.N.Cmp(expectedKey.N) != 0 {
		t.Error("Parsed key modulus doesn't match expected")
	}
	if parsedKey.E != expectedKey.E {
		t.Errorf("Parsed key exponent doesn't match expected: got %d, want %d", parsedKey.E, expectedKey.E)
	}
}

func TestParseRSAPublicKeyFromPEM_PKCS1(t *testing.T) {
	pemData, expectedKey := generateRSAPublicKeyPEM_PKCS1(t)

	parsedKey, err := ParseRSAPublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromPEM() failed: %v", err)
	}

	// Compare key parameters
	if parsedKey.N.Cmp(expectedKey.N) != 0 {
		t.Error("Parsed key modulus doesn't match expected")
	}
	if parsedKey.E != expectedKey.E {
		t.Errorf("Parsed key exponent doesn't match expected: got %d, want %d", parsedKey.E, expectedKey.E)
	}
}

func TestParseRSAPublicKeyFromPEM_InvalidPEM(t *testing.T) {
	invalidPEM := []byte("not a valid PEM")

	_, err := ParseRSAPublicKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("ParseRSAPublicKeyFromPEM() should fail for invalid PEM")
	}
}

func TestParseRSAPublicKeyFromPEM_WrongKeyType(t *testing.T) {
	// Generate ECDSA key PEM
	ecdsaPEM, _ := generateECDSAPublicKeyPEM_PKIX(t)

	_, err := ParseRSAPublicKeyFromPEM(ecdsaPEM)
	if err == nil {
		t.Error("ParseRSAPublicKeyFromPEM() should fail when PEM contains ECDSA key")
	}
}

func TestParseRSAPublicKeyFromPEM_UnsupportedBlockType(t *testing.T) {
	pemData := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU7GzMA0GCSqGSIb3DQEBCwUAMBMxETAPBgNVBAMMCHRl
-----END CERTIFICATE-----`)

	_, err := ParseRSAPublicKeyFromPEM(pemData)
	if err == nil {
		t.Error("ParseRSAPublicKeyFromPEM() should fail for unsupported block type")
	}
}

func TestParseECDSAPublicKeyFromPEM_PKIX(t *testing.T) {
	pemData, expectedKey := generateECDSAPublicKeyPEM_PKIX(t)

	parsedKey, err := ParseECDSAPublicKeyFromPEM(pemData)
	if err != nil {
		t.Fatalf("ParseECDSAPublicKeyFromPEM() failed: %v", err)
	}

	// Compare key parameters
	if parsedKey.Curve != expectedKey.Curve {
		t.Error("Parsed key curve doesn't match expected")
	}
	if parsedKey.X.Cmp(expectedKey.X) != 0 {
		t.Error("Parsed key X coordinate doesn't match expected")
	}
	if parsedKey.Y.Cmp(expectedKey.Y) != 0 {
		t.Error("Parsed key Y coordinate doesn't match expected")
	}
}

func TestParseECDSAPublicKeyFromPEM_InvalidPEM(t *testing.T) {
	invalidPEM := []byte("not a valid PEM")

	_, err := ParseECDSAPublicKeyFromPEM(invalidPEM)
	if err == nil {
		t.Error("ParseECDSAPublicKeyFromPEM() should fail for invalid PEM")
	}
}

func TestParseECDSAPublicKeyFromPEM_WrongKeyType(t *testing.T) {
	// Generate RSA key PEM
	rsaPEM, _ := generateRSAPublicKeyPEM_PKIX(t)

	_, err := ParseECDSAPublicKeyFromPEM(rsaPEM)
	if err == nil {
		t.Error("ParseECDSAPublicKeyFromPEM() should fail when PEM contains RSA key")
	}
}

func TestParseECDSAPublicKeyFromPEM_UnsupportedBlockType(t *testing.T) {
	pemData := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHCgVZU7GzMA0GCSqGSIb3DQEBCwUAMBMxETAPBgNVBAMMCHRl
-----END CERTIFICATE-----`)

	_, err := ParseECDSAPublicKeyFromPEM(pemData)
	if err == nil {
		t.Error("ParseECDSAPublicKeyFromPEM() should fail for unsupported block type")
	}
}
