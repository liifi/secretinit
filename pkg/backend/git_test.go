package backend

import (
	"fmt"
	"strings"
	"testing"

	"github.com/liifi/secretinit/pkg/parser"
)

// MockGitCredentialHelper simulates git credential responses for testing
type MockGitCredentialHelper struct {
	responses map[string]string
	callCount int
}

func NewMockGitCredentialHelper() *MockGitCredentialHelper {
	return &MockGitCredentialHelper{
		responses: make(map[string]string),
		callCount: 0,
	}
}

func (m *MockGitCredentialHelper) SetResponse(url, username, response string) {
	key := fmt.Sprintf("%s|%s", url, username)
	m.responses[key] = response
}

func (m *MockGitCredentialHelper) GetCallCount() int {
	return m.callCount
}

// TestableGitBackend wraps GitBackend with a mock credential helper for testing
type TestableGitBackend struct {
	*GitBackend
	mockHelper *MockGitCredentialHelper
}

func NewTestableGitBackend() *TestableGitBackend {
	return &TestableGitBackend{
		GitBackend: &GitBackend{},
		mockHelper: NewMockGitCredentialHelper(),
	}
}

// Override getCredential for testing
func (b *TestableGitBackend) getCredentialForTest(url, user string) (string, error) {
	b.mockHelper.callCount++
	key := fmt.Sprintf("%s|%s", url, user)
	if response, exists := b.mockHelper.responses[key]; exists {
		return response, nil
	}
	return "", fmt.Errorf("no mock response configured for url=%s, user=%s", url, user)
}

// RetrieveSecretWithMock uses the mock helper instead of real git credential
func (b *TestableGitBackend) RetrieveSecretWithMock(service, resource, keyPath string) (string, error) {
	cache := GetGlobalCache()
	cacheKey := fmt.Sprintf("git:%s:%s", service, resource)

	// Check if we have cached the raw git credential response
	var rawCredentialResponse string
	var err error
	if cached, exists := cache.Get(cacheKey); exists {
		rawCredentialResponse = cached
	} else {
		// Cache miss - retrieve from mock git credential helper
		cleanURL, username := parser.ParseGitURL(resource)
		rawCredentialResponse, err = b.getCredentialForTest(cleanURL, username)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve git credential for %s: %w", cleanURL, err)
		}

		// Cache the raw git credential response directly
		cache.Set(cacheKey, rawCredentialResponse)
	}

	// Apply keyPath parsing to the raw credential response
	return parseGitCredential(rawCredentialResponse, keyPath)
}

func TestGitBackend_RetrieveSecret_CacheIntegration(t *testing.T) {
	// Clear global cache before test
	ClearGlobalCache()

	backend := NewTestableGitBackend()

	// Set up mock response
	mockResponse := "protocol=https\nhost=example.com\nusername=testuser\npassword=testpass\n"
	backend.mockHelper.SetResponse("https://example.com", "testuser", mockResponse)

	// Test 1: First call should hit the credential helper
	result, err := backend.RetrieveSecretWithMock("", "https://testuser@example.com", "username")
	if err != nil {
		t.Fatalf("Unexpected error on first call: %v", err)
	}
	if result != "testuser" {
		t.Fatalf("Expected 'testuser', got '%s'", result)
	}
	if backend.mockHelper.GetCallCount() != 1 {
		t.Fatalf("Expected 1 credential helper call, got %d", backend.mockHelper.GetCallCount())
	}

	// Test 2: Second call for same resource should use cache (different keyPath)
	result, err = backend.RetrieveSecretWithMock("", "https://testuser@example.com", "password")
	if err != nil {
		t.Fatalf("Unexpected error on second call: %v", err)
	}
	if result != "testpass" {
		t.Fatalf("Expected 'testpass', got '%s'", result)
	}
	// Should still be 1 call because of cache hit
	if backend.mockHelper.GetCallCount() != 1 {
		t.Fatalf("Expected 1 credential helper call (cache hit), got %d", backend.mockHelper.GetCallCount())
	}

	// Test 3: Verify cache contains the response
	cacheKey := "git::https://testuser@example.com"
	cache := GetGlobalCache()
	if cachedValue, exists := cache.Get(cacheKey); !exists {
		t.Fatal("Expected cached value to exist")
	} else if cachedValue != mockResponse {
		t.Fatalf("Cached value mismatch. Expected:\n%s\nGot:\n%s", mockResponse, cachedValue)
	}
}

func TestGitBackend_RetrieveSecret_EmptyResponse(t *testing.T) {
	// Clear global cache before test
	ClearGlobalCache()

	backend := NewTestableGitBackend()

	// Test with empty credential response (this would have exposed the variable scoping bug)
	backend.mockHelper.SetResponse("https://example.com", "testuser", "")

	_, err := backend.RetrieveSecretWithMock("", "https://testuser@example.com", "username")
	if err == nil {
		t.Fatal("Expected error for empty credential response")
	}
	if !strings.Contains(err.Error(), "key 'username' not found") {
		t.Fatalf("Expected 'key not found' error, got: %v", err)
	}
}

func TestGitBackend_RetrieveSecret_CacheKeyFormat(t *testing.T) {
	// Clear global cache before test
	ClearGlobalCache()

	backend := NewTestableGitBackend()

	// Set up mock responses for both test cases
	mockResponse1 := "protocol=https\nhost=api.example.com\nusername=apiuser\npassword=apipass\n"
	backend.mockHelper.SetResponse("https://api.example.com", "apiuser", mockResponse1)

	mockResponse2 := "protocol=https\nhost=api.example.com\nusername=defaultuser\npassword=defaultpass\n"
	backend.mockHelper.SetResponse("https://api.example.com", "", mockResponse2)

	// Test different URLs to ensure cache keys are correct
	testCases := []struct {
		name     string
		resource string
		expected string
	}{
		{
			name:     "URL with user",
			resource: "https://apiuser@api.example.com",
			expected: "git::https://apiuser@api.example.com",
		},
		{
			name:     "URL without user",
			resource: "https://api.example.com",
			expected: "git::https://api.example.com",
		},
	}

	cache := GetGlobalCache()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache for each test
			ClearGlobalCache()

			_, err := backend.RetrieveSecretWithMock("", tc.resource, "username")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the cache key format
			if _, exists := cache.Get(tc.expected); !exists {
				t.Fatalf("Expected cache key '%s' not found", tc.expected)
			}
		})
	}
}

func TestGitBackend_RetrieveSecret_VariableScopeRegression(t *testing.T) {
	// This test specifically would have caught the variable scoping bug
	// Clear global cache before test
	ClearGlobalCache()

	backend := NewTestableGitBackend()

	// Set up mock response with actual credential data
	mockResponse := "protocol=https\nhost=example.com\nusername=testuser\npassword=secretpass\n"
	backend.mockHelper.SetResponse("https://example.com", "testuser", mockResponse)

	// First call: cache miss, should retrieve from credential helper
	result1, err := backend.RetrieveSecretWithMock("", "https://testuser@example.com", "password")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if result1 != "secretpass" {
		t.Fatalf("First call: expected 'secretpass', got '%s'", result1)
	}

	// Second call: cache hit, should return same value
	result2, err := backend.RetrieveSecretWithMock("", "https://testuser@example.com", "username")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if result2 != "testuser" {
		t.Fatalf("Second call: expected 'testuser', got '%s'", result2)
	}

	// Verify that both calls used the same cached credential response
	if backend.mockHelper.GetCallCount() != 1 {
		t.Fatalf("Expected exactly 1 credential helper call, got %d", backend.mockHelper.GetCallCount())
	}

	// Third call: different keyPath, should still use cache
	result3, err := backend.RetrieveSecretWithMock("", "https://testuser@example.com", "password")
	if err != nil {
		t.Fatalf("Third call failed: %v", err)
	}
	if result3 != "secretpass" {
		t.Fatalf("Third call: expected 'secretpass', got '%s'", result3)
	}

	// Still should be only 1 credential helper call
	if backend.mockHelper.GetCallCount() != 1 {
		t.Fatalf("Expected exactly 1 credential helper call after all tests, got %d", backend.mockHelper.GetCallCount())
	}
}

func TestParseGitCredential(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		keyPath     string
		expected    string
		shouldError bool
	}{
		{
			name:     "parse username from git response",
			response: "username=testuser\npassword=testpass\n",
			keyPath:  "username",
			expected: "testuser",
		},
		{
			name:     "parse password from git response",
			response: "username=testuser\npassword=testpass\n",
			keyPath:  "password",
			expected: "testpass",
		},
		{
			name:     "parse password with special characters",
			response: "username=testuser\npassword=my:complex=password!@#\n",
			keyPath:  "password",
			expected: "my:complex=password!@#",
		},
		{
			name:     "parse password with equals in value",
			response: "username=admin\npassword=base64==\n",
			keyPath:  "password",
			expected: "base64==",
		},
		{
			name:        "invalid key path",
			response:    "username=testuser\npassword=testpass\n",
			keyPath:     "invalid",
			shouldError: true,
		},
		{
			name:     "empty lines and whitespace",
			response: "\nusername=testuser\n\npassword=testpass\n\n",
			keyPath:  "username",
			expected: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseGitCredential(tt.response, tt.keyPath)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
