package jwt

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

// claimsContextKey is the context key for JWT claims.
// Using an unexported struct type prevents collisions with other packages.
type claimsContextKey struct{}

// WithClaims adds the JWT claims to the context.
//
// This is called internally by the middleware after successful validation.
func WithClaims(ctx context.Context, claims jwt.MapClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext retrieves the JWT claims from the context.
//
// Returns the claims and true if found, or nil and false if not found.
//
// Example:
//
//	claims, ok := jwt.ClaimsFromContext(ctx)
//	if !ok {
//	    // No claims in context (optional mode or public endpoint)
//	    return
//	}
//	userID := claims["sub"].(string)
func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(jwt.MapClaims)
	return claims, ok
}

// MustClaimsFromContext retrieves the JWT claims from the context.
//
// Panics if claims are not found. Use this only when you're certain
// that JWT validation has occurred (not in optional mode or skip paths).
//
// Example:
//
//	claims := jwt.MustClaimsFromContext(ctx)
//	userID := claims["sub"].(string)
func MustClaimsFromContext(ctx context.Context) jwt.MapClaims {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		panic("jwt: claims not found in context")
	}
	return claims
}

// SubjectFromContext retrieves the "sub" (subject) claim from the context.
//
// This is a convenience helper for the common use case of extracting the user ID.
//
// Returns the subject and true if found, or empty string and false if not found.
//
// Example:
//
//	userID, ok := jwt.SubjectFromContext(ctx)
//	if !ok {
//	    return errors.New("user not authenticated")
//	}
func SubjectFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}

	sub, ok := claims["sub"].(string)
	return sub, ok
}

// IssuerFromContext retrieves the "iss" (issuer) claim from the context.
//
// Returns the issuer and true if found, or empty string and false if not found.
//
// Example:
//
//	issuer, ok := jwt.IssuerFromContext(ctx)
func IssuerFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}

	iss, ok := claims["iss"].(string)
	return iss, ok
}
