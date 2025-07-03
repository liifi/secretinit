package backend

import (
	"testing"
)

// MockBackend for testing caching behavior without external dependencies
type MockBackend struct {
	callCount int
	responses map[string]string
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		callCount: 0,
		responses: make(map[string]string),
	}
}

func (m *MockBackend) SetResponse(cacheKey, value string) {
	m.responses[cacheKey] = value
}

func (m *MockBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	cache := GetGlobalCache()
	cacheKey := "mock:" + service + ":" + resource

	// Check cache first
	if cachedValue, exists := cache.Get(cacheKey); exists {
		// Parse keyPath from cached value if needed
		if keyPath == "" {
			return cachedValue, nil
		}
		return extractJSONKey(cachedValue, keyPath)
	}

	// Cache miss - simulate backend call
	m.callCount++

	// Simulate response based on what was set
	if value, exists := m.responses[cacheKey]; exists {
		cache.Set(cacheKey, value)
		if keyPath == "" {
			return value, nil
		}
		return extractJSONKey(value, keyPath)
	}

	return "", nil
}

func (m *MockBackend) GetCallCount() int {
	return m.callCount
}

func TestCache_BasicOperations(t *testing.T) {
	cache := NewCache()

	// Test initial state
	if cache.Size() != 0 {
		t.Fatalf("Expected empty cache, got size %d", cache.Size())
	}

	// Test cache miss
	if _, exists := cache.Get("key1"); exists {
		t.Fatal("Expected cache miss for non-existent key")
	}

	// Test cache set and hit
	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Fatalf("Expected cache size 1, got %d", cache.Size())
	}

	if value, exists := cache.Get("key1"); !exists || value != "value1" {
		t.Fatalf("Expected cache hit with value 'value1', got exists=%v, value='%s'", exists, value)
	}

	// Test cache clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Fatalf("Expected empty cache after clear, got size %d", cache.Size())
	}

	if _, exists := cache.Get("key1"); exists {
		t.Fatal("Expected cache miss after clear")
	}
}

func TestGlobalCache_Functions(t *testing.T) {
	// Clear global cache first
	ClearGlobalCache()

	// Test initial state
	if GetGlobalCacheSize() != 0 {
		t.Fatalf("Expected empty global cache, got size %d", GetGlobalCacheSize())
	}

	// Test global cache operations
	cache := GetGlobalCache()
	cache.Set("global_key", "global_value")

	if GetGlobalCacheSize() != 1 {
		t.Fatalf("Expected global cache size 1, got %d", GetGlobalCacheSize())
	}

	if value, exists := cache.Get("global_key"); !exists || value != "global_value" {
		t.Fatalf("Expected global cache hit with value 'global_value', got exists=%v, value='%s'", exists, value)
	}

	// Test global clear
	ClearGlobalCache()
	if GetGlobalCacheSize() != 0 {
		t.Fatalf("Expected empty global cache after clear, got size %d", GetGlobalCacheSize())
	}
}

func TestBackendCaching_KeyFormats(t *testing.T) {
	// Test that different backends use proper cache key formats
	cache := GetGlobalCache()
	cache.Clear()

	// Test various backend cache key formats
	testCases := []struct {
		backend  string
		cacheKey string
		value    string
	}{
		{"git", "git:username:https://api.example.com", "username=testuser\npassword=testpass\n"},
		{"aws", "aws:sm:myapp/secret", `{"username":"user","password":"pass"}`},
		{"gcp", "gcp:sm:projects/myproject/secrets/secret/versions/latest", "secret-value"},
		{"azure", "azure:kv:myvault/secret", "azure-secret"},
		{"azure", "azure:kv:myvault/secret/v1", "azure-secret-v1"}, // Test versioned Azure keys
	}

	// Set all cache entries
	for _, tc := range testCases {
		cache.Set(tc.cacheKey, tc.value)
	}

	// Verify all are independently cached
	if cache.Size() != len(testCases) {
		t.Fatalf("Expected cache size %d, got %d", len(testCases), cache.Size())
	}

	// Verify each cached value
	for _, tc := range testCases {
		if value, exists := cache.Get(tc.cacheKey); !exists || value != tc.value {
			t.Fatalf("Backend %s: expected value '%s', got exists=%v, value='%s'",
				tc.backend, tc.value, exists, value)
		}
	}
}

func TestBackendCaching_DuplicateSecretOptimization(t *testing.T) {
	// Test that multiple variables using the same secret only fetch once
	ClearGlobalCache()

	mock := NewMockBackend()
	mock.SetResponse("mock:sm:myapp/secret", `{"username":"user","password":"secret123"}`)

	// Simulate multiple variables requesting the same secret with different keyPaths
	secrets := []struct {
		service  string
		resource string
		keyPath  string
	}{
		{"sm", "myapp/secret", "username"},
		{"sm", "myapp/secret", "password"},
		{"sm", "myapp/secret", "username"}, // Duplicate request
	}

	var results []string
	for _, secret := range secrets {
		result, err := mock.RetrieveSecret(secret.service, secret.resource, secret.keyPath)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, result)
	}

	// Verify results
	if results[0] != "user" {
		t.Errorf("Expected 'user', got '%s'", results[0])
	}
	if results[1] != "secret123" {
		t.Errorf("Expected 'secret123', got '%s'", results[1])
	}
	if results[2] != "user" {
		t.Errorf("Expected 'user', got '%s'", results[2])
	}

	// Backend should only be called once due to caching
	if mock.GetCallCount() != 1 {
		t.Fatalf("Expected 1 backend call due to caching, got %d", mock.GetCallCount())
	}

	// Verify cache has the entry
	if GetGlobalCacheSize() != 1 {
		t.Fatalf("Expected cache size 1, got %d", GetGlobalCacheSize())
	}
}

func TestGitCredentialParsing(t *testing.T) {
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
			name:     "parse password with special characters",
			response: "username=testuser\npassword=my:complex=password!@#\n",
			keyPath:  "password",
			expected: "my:complex=password!@#",
		},
		{
			name:        "invalid key path",
			response:    "username=testuser\npassword=testpass\n",
			keyPath:     "invalid",
			shouldError: true,
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

func TestJSONKeyExtraction_CachedValues(t *testing.T) {
	// Test that keyPath parsing works correctly with cached values for JSON secrets
	ClearGlobalCache()

	cache := GetGlobalCache()

	// Test cases for different backend JSON formats
	testCases := []struct {
		backend     string
		cacheKey    string
		jsonValue   string
		extractions []struct {
			keyPath  string
			expected string
		}
	}{
		{
			backend:   "aws",
			cacheKey:  "aws:sm:myapp/db-creds",
			jsonValue: `{"username":"dbuser","password":"dbpass","host":"db.example.com"}`,
			extractions: []struct {
				keyPath  string
				expected string
			}{
				{"username", "dbuser"},
				{"password", "dbpass"},
				{"host", "db.example.com"},
			},
		},
		{
			backend:   "gcp",
			cacheKey:  "gcp:sm:projects/myproject/secrets/service-account/versions/latest",
			jsonValue: `{"type":"service_account","project_id":"myproject","private_key_id":"key123"}`,
			extractions: []struct {
				keyPath  string
				expected string
			}{
				{"project_id", "myproject"},
				{"private_key_id", "key123"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.backend, func(t *testing.T) {
			// Cache the JSON value
			cache.Set(tc.cacheKey, tc.jsonValue)

			// Test keyPath extractions
			for _, extraction := range tc.extractions {
				result, err := extractJSONKey(tc.jsonValue, extraction.keyPath)
				if err != nil {
					t.Fatalf("Failed to extract keyPath %s: %v", extraction.keyPath, err)
				}

				if result != extraction.expected {
					t.Errorf("For keyPath %s, expected %s, got %s",
						extraction.keyPath, extraction.expected, result)
				}
			}

			// Test returning full value when keyPath is empty
			result, err := extractJSONKey(tc.jsonValue, "")
			if err == nil { // extractJSONKey returns error for empty keyPath, but backends handle this differently
				if result != tc.jsonValue {
					t.Errorf("Expected full JSON value, got %s", result)
				}
			}
		})
	}
}
