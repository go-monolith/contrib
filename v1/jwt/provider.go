package jwt

import (
	"context"
)

// KeyProvider provides cryptographic keys for JWT signature verification.
//
// Implementations can provide keys from various sources:
//   - Static keys (StaticKeyProvider)
//   - JWKS endpoints (JWKSKeyProvider)
//   - Custom key sources
type KeyProvider interface {
	// GetKey retrieves the key for the given key ID (kid).
	//
	// For static key providers, kid is ignored.
	// For JWKS providers, kid is used to lookup the key in the key set.
	//
	// Returns the key ([]byte for HMAC, crypto.PublicKey for RSA/ECDSA) or an error.
	GetKey(ctx context.Context, kid string) (interface{}, error)
}

// StaticKeyProvider provides a single static key for JWT validation.
//
// This is used for HMAC secrets or pre-configured RSA/ECDSA public keys.
type StaticKeyProvider struct {
	key interface{} // []byte for HMAC, crypto.PublicKey for RSA/ECDSA
}

// NewStaticKeyProvider creates a new static key provider.
//
// The key should be:
//   - []byte for HMAC (HS256/HS384/HS512)
//   - *rsa.PublicKey for RSA (RS256/RS384/RS512)
//   - *ecdsa.PublicKey for ECDSA (ES256/ES384/ES512)
func NewStaticKeyProvider(key interface{}) *StaticKeyProvider {
	return &StaticKeyProvider{key: key}
}

// GetKey returns the static key, ignoring the kid parameter.
//
// For static key providers, the kid parameter is ignored since there is only
// one key configured.
func (p *StaticKeyProvider) GetKey(ctx context.Context, kid string) (interface{}, error) {
	return p.key, nil
}

// SecretProviderKeyProvider provides dynamic secrets based on the issuer claim.
//
// This is used for multi-tenant scenarios where different issuers use different
// HMAC secrets. The secret is looked up by calling the provider function with
// the issuer claim from the JWT.
//
// Note: This provider requires the issuer to be passed via the context or
// extracted from the token during validation.
type SecretProviderKeyProvider struct {
	provider SecretProvider
}

// NewSecretProviderKeyProvider creates a new secret provider key provider.
func NewSecretProviderKeyProvider(provider SecretProvider) *SecretProviderKeyProvider {
	return &SecretProviderKeyProvider{
		provider: provider,
	}
}

// GetKey retrieves the secret for the given issuer.
//
// For SecretProviderKeyProvider, the kid parameter is repurposed to pass the issuer.
// The validator extracts the issuer from the unverified token claims and passes
// it as the kid parameter.
func (p *SecretProviderKeyProvider) GetKey(ctx context.Context, issuer string) (interface{}, error) {
	// If issuer is empty, return error
	if issuer == "" {
		return nil, ErrMissingIssuer
	}

	// Get secret from provider
	secret, err := p.provider(issuer)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
