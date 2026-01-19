package jwt

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// ParseRSAPublicKeyFromPEM parses an RSA public key from PEM-encoded data.
//
// Supports both PKCS#1 (RSA PUBLIC KEY) and PKIX (PUBLIC KEY) formats.
//
// Example:
//
//	pemData := []byte(`-----BEGIN PUBLIC KEY-----
//	MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
//	-----END PUBLIC KEY-----`)
//	publicKey, err := ParseRSAPublicKeyFromPEM(pemData)
func ParseRSAPublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKIX format (PUBLIC KEY) first
	if block.Type == "PUBLIC KEY" {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}

		rsaKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA public key, got %T", pub)
		}

		return rsaKey, nil
	}

	// Try PKCS#1 format (RSA PUBLIC KEY)
	if block.Type == "RSA PUBLIC KEY" {
		rsaKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS#1 public key: %w", err)
		}

		return rsaKey, nil
	}

	return nil, fmt.Errorf("unsupported PEM block type: %s (expected PUBLIC KEY or RSA PUBLIC KEY)", block.Type)
}

// ParseECDSAPublicKeyFromPEM parses an ECDSA public key from PEM-encoded data.
//
// Supports PKIX (PUBLIC KEY) format.
//
// Example:
//
//	pemData := []byte(`-----BEGIN PUBLIC KEY-----
//	MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
//	-----END PUBLIC KEY-----`)
//	publicKey, err := ParseECDSAPublicKeyFromPEM(pemData)
func ParseECDSAPublicKeyFromPEM(pemData []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKIX format (PUBLIC KEY)
	if block.Type == "PUBLIC KEY" {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}

		ecdsaKey, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not an ECDSA public key, got %T", pub)
		}

		return ecdsaKey, nil
	}

	// Try EC PRIVATE KEY block (some systems export public key from private key format)
	if block.Type == "EC PUBLIC KEY" {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse EC public key: %w", err)
		}

		ecdsaKey, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not an ECDSA public key, got %T", pub)
		}

		return ecdsaKey, nil
	}

	return nil, fmt.Errorf("unsupported PEM block type: %s (expected PUBLIC KEY or EC PUBLIC KEY)", block.Type)
}
