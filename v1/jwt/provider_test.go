package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestStaticKeyProvider_HMACSecret(t *testing.T) {
	secret := []byte("test-secret-key-for-hmac")
	provider := NewStaticKeyProvider(secret)

	ctx := context.Background()
	key, err := provider.GetKey(ctx, "")
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	// Verify the returned key is the same as the secret
	keyBytes, ok := key.([]byte)
	if !ok {
		t.Fatalf("Expected key to be []byte, got %T", key)
	}

	if string(keyBytes) != string(secret) {
		t.Errorf("Expected key '%s', got '%s'", secret, keyBytes)
	}
}

func TestStaticKeyProvider_RSAPublicKey(t *testing.T) {
	// Generate test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	provider := NewStaticKeyProvider(publicKey)

	ctx := context.Background()
	key, err := provider.GetKey(ctx, "")
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	// Verify the returned key is an RSA public key
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("Expected key to be *rsa.PublicKey, got %T", key)
	}

	// Verify it's the same key
	if rsaKey.N.Cmp(publicKey.N) != 0 || rsaKey.E != publicKey.E {
		t.Error("Returned RSA key doesn't match the original")
	}
}

func TestStaticKeyProvider_ECDSAPublicKey(t *testing.T) {
	// Generate test ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	provider := NewStaticKeyProvider(publicKey)

	ctx := context.Background()
	key, err := provider.GetKey(ctx, "")
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	// Verify the returned key is an ECDSA public key
	ecdsaKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Expected key to be *ecdsa.PublicKey, got %T", key)
	}

	// Verify it's the same key
	if ecdsaKey.X.Cmp(publicKey.X) != 0 || ecdsaKey.Y.Cmp(publicKey.Y) != 0 {
		t.Error("Returned ECDSA key doesn't match the original")
	}
}

func TestStaticKeyProvider_KidParameterIgnored(t *testing.T) {
	secret := []byte("test-secret")
	provider := NewStaticKeyProvider(secret)

	ctx := context.Background()

	// Call GetKey with various kid values
	kidValues := []string{"", "kid-1", "kid-2", "any-kid-value"}

	for _, kid := range kidValues {
		key, err := provider.GetKey(ctx, kid)
		if err != nil {
			t.Fatalf("GetKey() with kid='%s' failed: %v", kid, err)
		}

		// All calls should return the same key
		keyBytes, ok := key.([]byte)
		if !ok {
			t.Fatalf("Expected key to be []byte, got %T", key)
		}

		if string(keyBytes) != string(secret) {
			t.Errorf("GetKey() with kid='%s' returned wrong key", kid)
		}
	}
}

func TestStaticKeyProvider_ContextNotUsed(t *testing.T) {
	secret := []byte("test-secret")
	provider := NewStaticKeyProvider(secret)

	// Call GetKey with different contexts
	contexts := []context.Context{
		context.Background(),
		context.TODO(),
	}

	for _, ctx := range contexts {
		key, err := provider.GetKey(ctx, "")
		if err != nil {
			t.Fatalf("GetKey() failed: %v", err)
		}

		// All calls should return the same key
		keyBytes, ok := key.([]byte)
		if !ok {
			t.Fatalf("Expected key to be []byte, got %T", key)
		}

		if string(keyBytes) != string(secret) {
			t.Error("GetKey() returned wrong key with different context")
		}
	}
}

func TestNewStaticKeyProvider_NilKey(t *testing.T) {
	// Test that provider can be created with nil key
	// (validation happens elsewhere, provider just stores what it's given)
	provider := NewStaticKeyProvider(nil)
	if provider == nil {
		t.Fatal("NewStaticKeyProvider() returned nil")
	}

	ctx := context.Background()
	key, err := provider.GetKey(ctx, "")
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	if key != nil {
		t.Errorf("Expected nil key, got %v", key)
	}
}
