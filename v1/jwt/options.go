package jwt

import (
	"crypto"
	"time"
)

// Option is a functional option for configuring the JWT middleware.
type Option func(*Config)

// WithSecret configures the middleware to use HMAC validation with the given secret.
//
// This is mutually exclusive with WithSecretProvider, WithPublicKey, and WithJWKSEndpoint.
//
// Example:
//
//	jwt.New(jwt.WithSecret([]byte("my-secret-key")))
func WithSecret(secret []byte) Option {
	return func(cfg *Config) {
		cfg.Secret = secret
	}
}

// WithSecretProvider configures the middleware to use a dynamic secret provider function.
//
// The provider function receives the issuer claim from the JWT and returns the
// corresponding secret for validation. This is useful for multi-tenant scenarios
// where different issuers use different secrets.
//
// This is mutually exclusive with WithSecret, WithPublicKey, and WithJWKSEndpoint.
//
// Example:
//
//	secretStore := map[string][]byte{
//	    "tenant-1": []byte("secret-1"),
//	    "tenant-2": []byte("secret-2"),
//	}
//	provider := func(issuer string) ([]byte, error) {
//	    secret, ok := secretStore[issuer]
//	    if !ok {
//	        return nil, fmt.Errorf("unknown issuer: %s", issuer)
//	    }
//	    return secret, nil
//	}
//	jwt.New(jwt.WithSecretProvider(provider))
func WithSecretProvider(provider SecretProvider) Option {
	return func(cfg *Config) {
		cfg.SecretProvider = provider
	}
}

// WithPublicKey configures the middleware to use RSA/ECDSA validation with the given public key.
//
// This is mutually exclusive with WithSecret and WithJWKSEndpoint.
//
// Example:
//
//	jwt.New(jwt.WithPublicKey(rsaPublicKey))
func WithPublicKey(publicKey crypto.PublicKey) Option {
	return func(cfg *Config) {
		cfg.PublicKey = publicKey
	}
}

// WithJWKSEndpoint configures the middleware to fetch public keys from a JWKS endpoint.
//
// This is mutually exclusive with WithSecret and WithPublicKey.
//
// Example:
//
//	jwt.New(jwt.WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"))
func WithJWKSEndpoint(endpoint string) Option {
	return func(cfg *Config) {
		cfg.JWKSEndpoint = endpoint
	}
}

// WithJWKSCacheTTL sets the duration to cache JWKS keys.
//
// Default: 1 hour
//
// Example:
//
//	jwt.New(
//	    jwt.WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
//	    jwt.WithJWKSCacheTTL(2 * time.Hour),
//	)
func WithJWKSCacheTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.JWKSCacheTTL = ttl
	}
}

// WithJWKSRefreshInterval sets the interval for background JWKS refresh.
//
// If set to 0, background refresh is disabled.
// Default: 50 minutes
//
// Example:
//
//	jwt.New(
//	    jwt.WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
//	    jwt.WithJWKSRefreshInterval(45 * time.Minute),
//	)
func WithJWKSRefreshInterval(interval time.Duration) Option {
	return func(cfg *Config) {
		cfg.JWKSRefreshInterval = interval
	}
}

// WithExpectedIssuer sets the expected issuer ("iss" claim) for JWT validation.
//
// If not set, issuer validation is skipped.
//
// Example:
//
//	jwt.New(
//	    jwt.WithSecret([]byte("my-secret")),
//	    jwt.WithExpectedIssuer("https://auth.example.com"),
//	)
func WithExpectedIssuer(issuer string) Option {
	return func(cfg *Config) {
		cfg.ExpectedIssuer = issuer
	}
}

// WithExpectedAudience sets the expected audience values ("aud" claim) for JWT validation.
//
// The token's aud claim must contain at least one of these values.
// If not set, audience validation is skipped.
//
// Example:
//
//	jwt.New(
//	    jwt.WithSecret([]byte("my-secret")),
//	    jwt.WithExpectedAudience("api-v1", "api-v2"),
//	)
func WithExpectedAudience(audience ...string) Option {
	return func(cfg *Config) {
		cfg.ExpectedAudience = audience
	}
}

// WithRequiredClaims sets the list of claims that must be present in the JWT.
//
// Example:
//
//	jwt.New(jwt.WithRequiredClaims("sub", "email", "role"))
func WithRequiredClaims(claims ...string) Option {
	return func(cfg *Config) {
		cfg.RequiredClaims = claims
	}
}

// WithAllowedAlgorithms sets the list of allowed signing algorithms.
//
// If not set, all algorithms matching the key type are allowed.
//
// Example:
//
//	jwt.New(jwt.WithAllowedAlgorithms("HS256", "HS384"))
func WithAllowedAlgorithms(algorithms ...string) Option {
	return func(cfg *Config) {
		cfg.AllowedAlgorithms = algorithms
	}
}

// WithClockSkew sets the clock skew tolerance for time-based claim validation.
//
// Default: 1 minute
//
// Example:
//
//	jwt.New(
//	    jwt.WithSecret([]byte("my-secret")),
//	    jwt.WithClockSkew(30 * time.Second),
//	)
func WithClockSkew(skew time.Duration) Option {
	return func(cfg *Config) {
		cfg.ClockSkew = skew
	}
}

// WithSkipPaths sets the service paths to skip JWT validation.
//
// Paths can be specified as "module.service.method", "module.service", or "module".
//
// Example:
//
//	jwt.New(jwt.WithSkipPaths("health.check", "metrics"))
func WithSkipPaths(paths ...string) Option {
	return func(cfg *Config) {
		cfg.SkipPaths = paths
	}
}

// WithOptional allows requests without tokens to proceed.
//
// When enabled, requests without Authorization headers will not be rejected,
// and claims will be nil in the context.
//
// Default: false
//
// Example:
//
//	jwt.New(
//	    jwt.WithSecret([]byte("my-secret")),
//	    jwt.WithOptional(true),
//	)
func WithOptional(optional bool) Option {
	return func(cfg *Config) {
		cfg.Optional = optional
	}
}
