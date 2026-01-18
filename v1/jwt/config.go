package jwt

import (
	"crypto"
	"time"
)

// SecretProvider is a function that returns the HMAC secret for a given issuer.
// Used for multi-tenant scenarios where different issuers use different secrets.
type SecretProvider func(issuer string) ([]byte, error)

// Config holds the JWT middleware configuration.
type Config struct {
	// Key sources (mutually exclusive - only ONE should be set)

	// Secret is the HMAC secret key for HS256/HS384/HS512 algorithms.
	Secret []byte

	// SecretProvider is a function that dynamically provides secrets based on the issuer.
	// Used for multi-tenant scenarios with per-issuer secrets.
	SecretProvider SecretProvider

	// PublicKey is the RSA/ECDSA public key for RS*/ES* algorithms.
	PublicKey crypto.PublicKey

	// JWKSEndpoint is the URL to fetch JWKS (JSON Web Key Set) for dynamic key fetching.
	JWKSEndpoint string

	// JWKS settings (only used when JWKSEndpoint is configured)

	// JWKSCacheTTL is the duration to cache JWKS keys before considering them stale.
	// Default: 1 hour
	JWKSCacheTTL time.Duration

	// JWKSRefreshInterval is the interval for background JWKS refresh.
	// If 0, background refresh is disabled.
	// Default: 50 minutes (should be less than JWKSCacheTTL)
	JWKSRefreshInterval time.Duration

	// JWKSRequestTimeout is the HTTP timeout for JWKS requests.
	// Default: 10 seconds
	JWKSRequestTimeout time.Duration

	// Validation settings

	// RequiredClaims is the list of claim names that must be present in the JWT.
	RequiredClaims []string

	// ExpectedIssuer is the expected value of the "iss" claim.
	// If empty, issuer validation is skipped.
	ExpectedIssuer string

	// ExpectedAudience is the list of acceptable values for the "aud" claim.
	// The token's aud claim must contain at least one of these values.
	// If empty, audience validation is skipped.
	ExpectedAudience []string

	// AllowedAlgorithms is the list of allowed signing algorithms (e.g., "HS256", "RS256").
	// If empty, all algorithms matching the key type are allowed.
	AllowedAlgorithms []string

	// ClockSkew is the duration to allow for clock drift when validating time-based claims.
	// Default: 1 minute
	ClockSkew time.Duration

	// Header extraction settings

	// HeaderKey is the header key to extract the JWT from.
	// Default: "authorization"
	HeaderKey string

	// TokenPrefix is the prefix before the token in the header value.
	// Default: "Bearer "
	TokenPrefix string

	// Behavior settings

	// SkipPaths is the list of service paths to skip JWT validation.
	// Paths can be: "module.service", "module", or "service"
	SkipPaths []string

	// Optional allows requests without tokens to proceed (claims will be nil in context).
	// Default: false
	Optional bool
}

// validateConfig validates the configuration and returns an error if invalid.
func validateConfig(cfg *Config) error {
	// Count configured key sources
	keySources := 0
	if len(cfg.Secret) > 0 {
		keySources++
	}
	if cfg.SecretProvider != nil {
		keySources++
	}
	if cfg.PublicKey != nil {
		keySources++
	}
	if cfg.JWKSEndpoint != "" {
		keySources++
	}

	// Ensure exactly one key source is configured
	if keySources == 0 {
		return ErrNoKeySourceConfigured
	}
	if keySources > 1 {
		return ErrMultipleKeySourcesConfigured
	}

	// Validate JWKS settings if endpoint is configured
	if cfg.JWKSEndpoint != "" {
		if cfg.JWKSCacheTTL <= 0 {
			cfg.JWKSCacheTTL = 1 * time.Hour // Apply default
		}
		if cfg.JWKSRequestTimeout <= 0 {
			cfg.JWKSRequestTimeout = 10 * time.Second // Apply default
		}
	}

	return nil
}

// applyDefaults applies default values to the configuration.
func applyDefaults(cfg *Config) {
	if cfg.JWKSCacheTTL == 0 {
		cfg.JWKSCacheTTL = 1 * time.Hour
	}

	if cfg.JWKSRefreshInterval == 0 && cfg.JWKSEndpoint != "" {
		cfg.JWKSRefreshInterval = 50 * time.Minute
	}

	if cfg.JWKSRequestTimeout == 0 {
		cfg.JWKSRequestTimeout = 10 * time.Second
	}

	if cfg.ClockSkew == 0 {
		cfg.ClockSkew = 1 * time.Minute
	}

	if cfg.HeaderKey == "" {
		cfg.HeaderKey = "authorization"
	}

	if cfg.TokenPrefix == "" {
		cfg.TokenPrefix = "Bearer "
	}
}
