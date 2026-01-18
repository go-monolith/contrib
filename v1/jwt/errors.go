package jwt

import "errors"

// Error types for JWT validation failures.
// These errors provide clear, descriptive messages for different validation failures.
var (
	// ErrMissingAuthHeader is returned when the Authorization header is missing.
	ErrMissingAuthHeader = errors.New("missing authorization header")

	// ErrInvalidAuthHeader is returned when the Authorization header format is invalid.
	// Expected format: "Bearer <token>"
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")

	// ErrInvalidToken is returned when the JWT token cannot be parsed.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when the token's exp claim is in the past.
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenNotYetValid is returned when the token's nbf claim is in the future.
	ErrTokenNotYetValid = errors.New("token not yet valid")

	// ErrInvalidSignature is returned when the JWT signature verification fails.
	ErrInvalidSignature = errors.New("invalid token signature")

	// ErrInvalidIssuer is returned when the iss claim doesn't match the expected issuer.
	ErrInvalidIssuer = errors.New("invalid issuer")

	// ErrMissingIssuer is returned when the iss claim is missing (required for secret provider).
	ErrMissingIssuer = errors.New("missing issuer claim")

	// ErrInvalidAudience is returned when the aud claim doesn't contain any expected audience.
	ErrInvalidAudience = errors.New("invalid audience")

	// ErrInvalidIssuedAt is returned when the iat claim is in the future.
	ErrInvalidIssuedAt = errors.New("invalid issued at time")

	// ErrInvalidClaims is returned when the claims format is invalid.
	ErrInvalidClaims = errors.New("invalid claims format")

	// ErrMissingRequiredClaim is returned when a required claim is missing.
	ErrMissingRequiredClaim = errors.New("missing required claim")

	// ErrJWKSFetchFailed is returned when fetching JWKS from the endpoint fails.
	ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")

	// ErrUnsupportedAlgorithm is returned when the JWT algorithm is not in the allowed list.
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")

	// ErrNoKeySourceConfigured is returned when no key source (Secret, PublicKey, or JWKS) is configured.
	ErrNoKeySourceConfigured = errors.New("no key source configured")

	// ErrMultipleKeySourcesConfigured is returned when multiple key sources are configured (mutually exclusive).
	ErrMultipleKeySourcesConfigured = errors.New("multiple key sources configured")

	// ErrKeyNotFound is returned when the requested key ID (kid) is not found in JWKS.
	ErrKeyNotFound = errors.New("key not found")
)
