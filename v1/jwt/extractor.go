package jwt

import (
	"strings"
)

// extractToken extracts a JWT token from message headers.
//
// It performs case-insensitive lookup for the header key and validates
// the header format to match: "<prefix> <token>".
//
// Parameters:
//   - headers: Map of message headers (keys are case-insensitive)
//   - headerKey: The name of the header to extract from (e.g., "authorization")
//   - tokenPrefix: The prefix before the token (e.g., "Bearer ")
//
// Returns:
//   - The extracted token string
//   - ErrMissingAuthHeader if the header is not found
//   - ErrInvalidAuthHeader if the header format is invalid
//
// Example:
//
//	headers := map[string][]string{
//	    "Authorization": {"Bearer eyJhbGciOiJIUzI1NiIs..."},
//	}
//	token, err := extractToken(headers, "authorization", "Bearer ")
func extractToken(headers map[string][]string, headerKey, tokenPrefix string) (string, error) {
	// Perform header lookup with optimization
	var headerValue string

	// Try exact match first (O(1) - most common case)
	if values, ok := headers[headerKey]; ok && len(values) > 0 {
		headerValue = values[0]
	} else {
		// Fall back to case-insensitive search (O(n))
		for key, values := range headers {
			if strings.EqualFold(key, headerKey) {
				if len(values) > 0 {
					headerValue = values[0]
				}
				break
			}
		}
	}

	// Check if header was found
	if headerValue == "" {
		return "", ErrMissingAuthHeader
	}

	// Validate header format: "<prefix> <token>"
	// Support case-insensitive prefix matching
	if !strings.HasPrefix(strings.ToLower(headerValue), strings.ToLower(tokenPrefix)) {
		return "", ErrInvalidAuthHeader
	}

	// Extract token by removing the prefix
	token := headerValue[len(tokenPrefix):]
	token = strings.TrimSpace(token)

	// Ensure token is not empty after removing prefix
	if token == "" {
		return "", ErrInvalidAuthHeader
	}

	return token, nil
}
