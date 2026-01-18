package http

import "strings"

// isUnauthorizedError checks if an error indicates missing or invalid JWT.
func isUnauthorizedError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no JWT claims") ||
		strings.Contains(msg, "missing 'sub' claim") ||
		strings.Contains(msg, "'sub' claim is") ||
		strings.Contains(msg, "invalid token") ||
		strings.Contains(msg, "missing authorization header") ||
		strings.Contains(msg, "authorization header")
}

// isForbiddenError checks if an error indicates authorization failure.
func isForbiddenError(err error) bool {
	return strings.Contains(err.Error(), "forbidden")
}

// isNotFoundError checks if an error indicates a resource was not found.
func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

// isValidationError checks if an error indicates a validation failure.
func isValidationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "is required") ||
		strings.Contains(msg, "cannot exceed") ||
		strings.Contains(msg, "cannot be empty") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "at least one field")
}
