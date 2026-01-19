package jwt

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultAllowedAlgorithms is the default list of allowed signing algorithms
// when AllowedAlgorithms is not configured. This prevents algorithm confusion
// attacks by explicitly excluding dangerous algorithms like "none".
var DefaultAllowedAlgorithms = []string{
	"HS256", "HS384", "HS512",
	"RS256", "RS384", "RS512",
	"ES256", "ES384", "ES512",
}

// TokenValidator validates JWT tokens using a configured key provider.
type TokenValidator struct {
	keyProvider KeyProvider
	config      *Config
}

// NewTokenValidator creates a new token validator with the given key provider and config.
func NewTokenValidator(keyProvider KeyProvider, config *Config) *TokenValidator {
	return &TokenValidator{
		keyProvider: keyProvider,
		config:      config,
	}
}

// Validate validates a JWT token string and returns the parsed claims.
//
// This method:
//   1. Parses the JWT token
//   2. Verifies the signature using the key provider
//   3. Validates the algorithm is allowed
//   4. Validates standard claims (exp, nbf, iat)
//   5. Validates issuer and audience (if configured)
//   6. Validates required claims (if configured)
//
// Special behavior for JWKS mode:
//   If signature validation fails and using JWKSKeyProvider, the validator
//   will automatically refresh the JWKS cache and retry once. This handles
//   the case where keys have been rotated.
//
// Returns the claims if validation succeeds, or an error if validation fails.
func (v *TokenValidator) Validate(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	// Try to parse and validate the token
	token, err := v.parseToken(ctx, tokenString)

	// Check for parsing errors
	if err != nil {
		// Check if it's a signature error and we're using JWKS
		if errors.Is(err, jwt.ErrSignatureInvalid) || (token != nil && !token.Valid) {
			// Check if provider is JWKSKeyProvider
			if jwksProvider, ok := v.keyProvider.(*JWKSKeyProvider); ok {
				// Refresh JWKS cache and retry once
				if refreshErr := jwksProvider.RefreshCache(ctx); refreshErr == nil {
					// Retry parsing with refreshed cache
					token, err = v.parseToken(ctx, tokenString)
					if err == nil && token.Valid {
						// Success after refresh - continue with validation
						goto validateClaims
					}
				}
			}
		}

		// Handle error (original or retry failure)
		if errors.Is(err, ErrUnsupportedAlgorithm) {
			return nil, ErrUnsupportedAlgorithm
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		if token != nil && !token.Valid {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}

validateClaims:
	// Verify the token is valid
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract claims as MapClaims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// Validate standard claims (exp, nbf, iat) with clock skew
	if err := v.validateStandardClaims(claims); err != nil {
		return nil, err
	}

	// Validate issuer if configured
	if err := v.validateIssuer(claims); err != nil {
		return nil, err
	}

	// Validate audience if configured
	if err := v.validateAudience(claims); err != nil {
		return nil, err
	}

	// Validate required claims if configured
	if err := v.validateRequiredClaims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// parseToken parses a JWT token and verifies its signature.
//
// This is a helper method used by Validate to parse the token.
// It can be called multiple times (e.g., for retry after JWKS refresh).
//
// Returns the parsed token and any parsing error.
func (v *TokenValidator) parseToken(ctx context.Context, tokenString string) (*jwt.Token, error) {
	// Parse the JWT token with custom key function and skip default time validation
	// We'll do our own time validation with clock skew
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check if algorithm is allowed
		alg := token.Method.Alg()
		if !v.isAlgorithmAllowed(alg) {
			return nil, ErrUnsupportedAlgorithm
		}

		// Determine the key identifier to use
		keyID := ""

		// For SecretProviderKeyProvider, use issuer as the key ID
		if _, ok := v.keyProvider.(*SecretProviderKeyProvider); ok {
			// Extract issuer from unverified claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if iss, ok := claims["iss"]; ok {
					if issStr, ok := iss.(string); ok {
						keyID = issStr
					}
				}
			}
			// If issuer is not found, keyID remains empty and will cause an error
		} else {
			// For other providers (JWKS), use kid from header
			if kidVal, ok := token.Header["kid"]; ok {
				if kidStr, ok := kidVal.(string); ok {
					keyID = kidStr
				}
			}
		}

		// Get the key from the provider
		key, err := v.keyProvider.GetKey(ctx, keyID)
		if err != nil {
			return nil, err
		}

		return key, nil
	})

	return token, err
}

// validateStandardClaims validates the standard JWT claims (exp, nbf, iat) with clock skew.
//
// Clock skew allows for small time differences between servers:
//   - exp: Token is valid until exp + clockSkew
//   - nbf: Token is valid from nbf - clockSkew
//   - iat: Token must be issued before now + clockSkew
func (v *TokenValidator) validateStandardClaims(claims jwt.MapClaims) error {
	now := time.Now()
	clockSkew := v.config.ClockSkew

	// Validate exp (expiration time)
	if exp, ok := claims["exp"]; ok {
		var expTime time.Time
		switch v := exp.(type) {
		case float64:
			expTime = time.Unix(int64(v), 0)
		case int64:
			expTime = time.Unix(v, 0)
		default:
			// If exp exists but is not a valid type, ignore it
			return nil
		}

		// Token is expired if current time > exp + clock skew
		if now.After(expTime.Add(clockSkew)) {
			return ErrTokenExpired
		}
	}

	// Validate nbf (not before)
	if nbf, ok := claims["nbf"]; ok {
		var nbfTime time.Time
		switch v := nbf.(type) {
		case float64:
			nbfTime = time.Unix(int64(v), 0)
		case int64:
			nbfTime = time.Unix(v, 0)
		default:
			// If nbf exists but is not a valid type, ignore it
			return nil
		}

		// Token is not yet valid if current time < nbf - clock skew
		if now.Before(nbfTime.Add(-clockSkew)) {
			return ErrTokenNotYetValid
		}
	}

	// Validate iat (issued at)
	if iat, ok := claims["iat"]; ok {
		var iatTime time.Time
		switch v := iat.(type) {
		case float64:
			iatTime = time.Unix(int64(v), 0)
		case int64:
			iatTime = time.Unix(v, 0)
		default:
			// If iat exists but is not a valid type, ignore it
			return nil
		}

		// Token was issued in the future if iat > now + clock skew
		if iatTime.After(now.Add(clockSkew)) {
			return ErrInvalidIssuedAt
		}
	}

	return nil
}

// validateIssuer validates the "iss" claim if ExpectedIssuer is configured.
//
// If ExpectedIssuer is not configured, this validation is skipped.
func (v *TokenValidator) validateIssuer(claims jwt.MapClaims) error {
	// Skip validation if no expected issuer is configured
	if v.config.ExpectedIssuer == "" {
		return nil
	}

	// Get issuer from claims
	iss, ok := claims["iss"]
	if !ok {
		return ErrInvalidIssuer
	}

	// Issuer must be a string
	issStr, ok := iss.(string)
	if !ok {
		return ErrInvalidIssuer
	}

	// Check if issuer matches expected value
	if issStr != v.config.ExpectedIssuer {
		return ErrInvalidIssuer
	}

	return nil
}

// validateAudience validates the "aud" claim if ExpectedAudience is configured.
//
// The aud claim can be a string or an array of strings.
// Validation succeeds if ANY expected audience is present in the token's aud claim.
//
// If ExpectedAudience is not configured, this validation is skipped.
func (v *TokenValidator) validateAudience(claims jwt.MapClaims) error {
	// Skip validation if no expected audience is configured
	if len(v.config.ExpectedAudience) == 0 {
		return nil
	}

	// Get audience from claims
	aud, ok := claims["aud"]
	if !ok {
		return ErrInvalidAudience
	}

	// Convert aud to string slice
	var audSlice []string
	switch v := aud.(type) {
	case string:
		audSlice = []string{v}
	case []interface{}:
		for _, a := range v {
			if audStr, ok := a.(string); ok {
				audSlice = append(audSlice, audStr)
			}
		}
	case []string:
		audSlice = v
	default:
		return ErrInvalidAudience
	}

	// Check if any expected audience is present in the token's audience
	for _, expected := range v.config.ExpectedAudience {
		for _, actual := range audSlice {
			if expected == actual {
				return nil // Found a match
			}
		}
	}

	// No match found
	return ErrInvalidAudience
}

// validateRequiredClaims validates that all required claims are present and non-empty.
//
// If RequiredClaims is not configured, this validation is skipped.
func (v *TokenValidator) validateRequiredClaims(claims jwt.MapClaims) error {
	// Skip validation if no required claims are configured
	if len(v.config.RequiredClaims) == 0 {
		return nil
	}

	// Check each required claim
	for _, requiredClaim := range v.config.RequiredClaims {
		value, ok := claims[requiredClaim]
		if !ok {
			return ErrMissingRequiredClaim
		}

		// Check if value is non-empty
		// Empty string, empty array, or nil are considered empty
		switch v := value.(type) {
		case string:
			if v == "" {
				return ErrMissingRequiredClaim
			}
		case []interface{}:
			if len(v) == 0 {
				return ErrMissingRequiredClaim
			}
		case []string:
			if len(v) == 0 {
				return ErrMissingRequiredClaim
			}
		case nil:
			return ErrMissingRequiredClaim
		}
		// For other types (numbers, booleans, objects), consider them valid
	}

	return nil
}

// isAlgorithmAllowed checks if the given algorithm is in the allowed list.
//
// If AllowedAlgorithms is empty, DefaultAllowedAlgorithms is used to prevent
// algorithm confusion attacks (e.g., "none" algorithm).
// If AllowedAlgorithms is configured, only those algorithms are allowed.
func (v *TokenValidator) isAlgorithmAllowed(alg string) bool {
	// Use configured algorithms or default to a secure set
	allowedList := v.config.AllowedAlgorithms
	if len(allowedList) == 0 {
		allowedList = DefaultAllowedAlgorithms
	}

	// Check if algorithm is in the allowed list
	for _, allowed := range allowedList {
		if alg == allowed {
			return true
		}
	}

	return false
}
