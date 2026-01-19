package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestValidateConfig_ValidSecret(t *testing.T) {
	cfg := &Config{
		Secret: []byte("test-secret"),
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() failed for valid secret config: %v", err)
	}
}

func TestValidateConfig_ValidPublicKey_RSA(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	cfg := &Config{
		PublicKey: &privateKey.PublicKey,
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() failed for valid RSA public key config: %v", err)
	}
}

func TestValidateConfig_ValidPublicKey_ECDSA(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	cfg := &Config{
		PublicKey: &privateKey.PublicKey,
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() failed for valid ECDSA public key config: %v", err)
	}
}

func TestValidateConfig_ValidJWKS(t *testing.T) {
	cfg := &Config{
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() failed for valid JWKS config: %v", err)
	}
}

func TestValidateConfig_NoKeySource(t *testing.T) {
	cfg := &Config{}

	err := validateConfig(cfg)
	if err != ErrNoKeySourceConfigured {
		t.Errorf("validateConfig() expected ErrNoKeySourceConfigured, got: %v", err)
	}
}

func TestValidateConfig_MultipleKeySources_SecretAndPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	cfg := &Config{
		Secret:    []byte("test-secret"),
		PublicKey: &privateKey.PublicKey,
	}

	err = validateConfig(cfg)
	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("validateConfig() expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}
}

func TestValidateConfig_MultipleKeySources_SecretAndJWKS(t *testing.T) {
	cfg := &Config{
		Secret:       []byte("test-secret"),
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
	}

	err := validateConfig(cfg)
	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("validateConfig() expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}
}

func TestValidateConfig_MultipleKeySources_PublicKeyAndJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	cfg := &Config{
		PublicKey:    &privateKey.PublicKey,
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
	}

	err = validateConfig(cfg)
	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("validateConfig() expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}
}

func TestValidateConfig_MultipleKeySources_All(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	cfg := &Config{
		Secret:       []byte("test-secret"),
		PublicKey:    &privateKey.PublicKey,
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
	}

	err = validateConfig(cfg)
	if err != ErrMultipleKeySourcesConfigured {
		t.Errorf("validateConfig() expected ErrMultipleKeySourcesConfigured, got: %v", err)
	}
}

func TestValidateConfig_JWKSDefaultValues(t *testing.T) {
	cfg := &Config{
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
		// Don't set JWKSCacheTTL or JWKSRequestTimeout
	}

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig() failed: %v", err)
	}

	// Verify defaults were applied
	if cfg.JWKSCacheTTL != 1*time.Hour {
		t.Errorf("Expected JWKSCacheTTL to be 1 hour, got: %v", cfg.JWKSCacheTTL)
	}
	if cfg.JWKSRequestTimeout != 10*time.Second {
		t.Errorf("Expected JWKSRequestTimeout to be 10 seconds, got: %v", cfg.JWKSRequestTimeout)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Secret: []byte("test-secret"),
	}

	applyDefaults(cfg)

	// Verify all defaults are applied
	if cfg.JWKSCacheTTL != 1*time.Hour {
		t.Errorf("Expected JWKSCacheTTL to be 1 hour, got: %v", cfg.JWKSCacheTTL)
	}
	if cfg.ClockSkew != 1*time.Minute {
		t.Errorf("Expected ClockSkew to be 1 minute, got: %v", cfg.ClockSkew)
	}
	if cfg.HeaderKey != "authorization" {
		t.Errorf("Expected HeaderKey to be 'authorization', got: %v", cfg.HeaderKey)
	}
	if cfg.TokenPrefix != "Bearer " {
		t.Errorf("Expected TokenPrefix to be 'Bearer ', got: %v", cfg.TokenPrefix)
	}
}

func TestApplyDefaults_JWKSRefreshInterval(t *testing.T) {
	cfg := &Config{
		JWKSEndpoint: "https://example.com/.well-known/jwks.json",
	}

	applyDefaults(cfg)

	// Verify JWKS refresh interval is set when endpoint is configured
	if cfg.JWKSRefreshInterval != 50*time.Minute {
		t.Errorf("Expected JWKSRefreshInterval to be 50 minutes, got: %v", cfg.JWKSRefreshInterval)
	}
}

func TestApplyDefaults_CustomValues(t *testing.T) {
	cfg := &Config{
		Secret:                []byte("test-secret"),
		JWKSCacheTTL:          2 * time.Hour,
		JWKSRefreshInterval:   90 * time.Minute,
		JWKSRequestTimeout:    30 * time.Second,
		ClockSkew:             5 * time.Minute,
		HeaderKey:             "x-auth-token",
		TokenPrefix:           "JWT ",
	}

	applyDefaults(cfg)

	// Verify custom values are not overwritten
	if cfg.JWKSCacheTTL != 2*time.Hour {
		t.Errorf("Expected JWKSCacheTTL to remain 2 hours, got: %v", cfg.JWKSCacheTTL)
	}
	if cfg.JWKSRefreshInterval != 90*time.Minute {
		t.Errorf("Expected JWKSRefreshInterval to remain 90 minutes, got: %v", cfg.JWKSRefreshInterval)
	}
	if cfg.JWKSRequestTimeout != 30*time.Second {
		t.Errorf("Expected JWKSRequestTimeout to remain 30 seconds, got: %v", cfg.JWKSRequestTimeout)
	}
	if cfg.ClockSkew != 5*time.Minute {
		t.Errorf("Expected ClockSkew to remain 5 minutes, got: %v", cfg.ClockSkew)
	}
	if cfg.HeaderKey != "x-auth-token" {
		t.Errorf("Expected HeaderKey to remain 'x-auth-token', got: %v", cfg.HeaderKey)
	}
	if cfg.TokenPrefix != "JWT " {
		t.Errorf("Expected TokenPrefix to remain 'JWT ', got: %v", cfg.TokenPrefix)
	}
}
