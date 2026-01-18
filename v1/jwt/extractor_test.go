package jwt

import (
	"testing"
)

func TestExtractToken_ValidHeader(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != nil {
		t.Fatalf("extractToken() failed: %v", err)
	}

	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	if token != expected {
		t.Errorf("Expected token '%s', got '%s'", expected, token)
	}
}

func TestExtractToken_CaseInsensitiveHeaderKey(t *testing.T) {
	testCases := []struct {
		name      string
		headerKey string
		headers   map[string][]string
	}{
		{
			name:      "lowercase authorization",
			headerKey: "authorization",
			headers: map[string][]string{
				"Authorization": {"Bearer token123"},
			},
		},
		{
			name:      "uppercase AUTHORIZATION",
			headerKey: "AUTHORIZATION",
			headers: map[string][]string{
				"authorization": {"Bearer token123"},
			},
		},
		{
			name:      "mixed case AuThOrIzAtIoN",
			headerKey: "AuThOrIzAtIoN",
			headers: map[string][]string{
				"authorization": {"Bearer token123"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := extractToken(tc.headers, tc.headerKey, "Bearer ")
			if err != nil {
				t.Fatalf("extractToken() failed: %v", err)
			}

			if token != "token123" {
				t.Errorf("Expected token 'token123', got '%s'", token)
			}
		})
	}
}

func TestExtractToken_CaseInsensitivePrefixMatching(t *testing.T) {
	testCases := []struct {
		name        string
		headerValue string
		prefix      string
		expected    string
	}{
		{
			name:        "lowercase bearer",
			headerValue: "bearer token123",
			prefix:      "Bearer ",
			expected:    "token123",
		},
		{
			name:        "uppercase BEARER",
			headerValue: "BEARER token123",
			prefix:      "Bearer ",
			expected:    "token123",
			},
		{
			name:        "mixed case BeArEr",
			headerValue: "BeArEr token123",
			prefix:      "Bearer ",
			expected:    "token123",
		},
		{
			name:        "uppercase prefix config",
			headerValue: "Bearer token123",
			prefix:      "BEARER ",
			expected:    "token123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string][]string{
				"Authorization": {tc.headerValue},
			}

			token, err := extractToken(headers, "authorization", tc.prefix)
			if err != nil {
				t.Fatalf("extractToken() failed: %v", err)
			}

			if token != tc.expected {
				t.Errorf("Expected token '%s', got '%s'", tc.expected, token)
			}
		})
	}
}

func TestExtractToken_MissingHeader(t *testing.T) {
	headers := map[string][]string{
		"Content-Type": {"application/json"},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != ErrMissingAuthHeader {
		t.Errorf("Expected ErrMissingAuthHeader, got: %v", err)
	}
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}
}

func TestExtractToken_EmptyHeaderValue(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != ErrMissingAuthHeader {
		t.Errorf("Expected ErrMissingAuthHeader, got: %v", err)
	}
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}
}

func TestExtractToken_InvalidFormat_NoPrefix(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != ErrInvalidAuthHeader {
		t.Errorf("Expected ErrInvalidAuthHeader, got: %v", err)
	}
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}
}

func TestExtractToken_InvalidFormat_WrongPrefix(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Basic dXNlcjpwYXNz"},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != ErrInvalidAuthHeader {
		t.Errorf("Expected ErrInvalidAuthHeader, got: %v", err)
	}
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}
}

func TestExtractToken_InvalidFormat_EmptyToken(t *testing.T) {
	testCases := []struct {
		name        string
		headerValue string
	}{
		{
			name:        "only prefix",
			headerValue: "Bearer ",
		},
		{
			name:        "prefix with whitespace only",
			headerValue: "Bearer   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string][]string{
				"Authorization": {tc.headerValue},
			}

			token, err := extractToken(headers, "authorization", "Bearer ")
			if err != ErrInvalidAuthHeader {
				t.Errorf("Expected ErrInvalidAuthHeader, got: %v", err)
			}
			if token != "" {
				t.Errorf("Expected empty token, got: %s", token)
			}
		})
	}
}

func TestExtractToken_MultipleHeaderValues(t *testing.T) {
	// When multiple values exist, take the first one
	headers := map[string][]string{
		"Authorization": {
			"Bearer first-token",
			"Bearer second-token",
		},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != nil {
		t.Fatalf("extractToken() failed: %v", err)
	}

	if token != "first-token" {
		t.Errorf("Expected token 'first-token', got '%s'", token)
	}
}

func TestExtractToken_CustomHeaderKey(t *testing.T) {
	headers := map[string][]string{
		"X-Auth-Token": {"Bearer custom-token"},
	}

	token, err := extractToken(headers, "x-auth-token", "Bearer ")
	if err != nil {
		t.Fatalf("extractToken() failed: %v", err)
	}

	if token != "custom-token" {
		t.Errorf("Expected token 'custom-token', got '%s'", token)
	}
}

func TestExtractToken_CustomPrefix(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"JWT my-custom-token"},
	}

	token, err := extractToken(headers, "authorization", "JWT ")
	if err != nil {
		t.Fatalf("extractToken() failed: %v", err)
	}

	if token != "my-custom-token" {
		t.Errorf("Expected token 'my-custom-token', got '%s'", token)
	}
}

func TestExtractToken_TokenWithSpaces(t *testing.T) {
	// Token value with extra whitespace should be trimmed
	headers := map[string][]string{
		"Authorization": {"Bearer   token-with-spaces   "},
	}

	token, err := extractToken(headers, "authorization", "Bearer ")
	if err != nil {
		t.Fatalf("extractToken() failed: %v", err)
	}

	expected := "token-with-spaces"
	if token != expected {
		t.Errorf("Expected token '%s', got '%s'", expected, token)
	}
}
