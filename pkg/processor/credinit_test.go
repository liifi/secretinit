package processor

import (
	"testing"
)

// MockGitBackend for testing
type MockGitBackend struct {
	username string
	password string
	err      error
}

func (m *MockGitBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	switch keyPath {
	case "username":
		return m.username, nil
	case "password":
		return m.password, nil
	default:
		return "", nil
	}
}

func TestCredInitProcessor_ProcessCredInitSecrets(t *testing.T) {
	tests := []struct {
		name        string
		secretVars  map[string]string
		mockBackend *MockGitBackend
		expected    map[string]string
		expectError bool
	}{
		{
			name: "Multi-credential mode - no keyPath",
			secretVars: map[string]string{
				"M": "git:https://test@api.example.com",
			},
			mockBackend: &MockGitBackend{
				username: "test",
				password: "testpass",
			}, expected: map[string]string{
				"M":      "secretinit:git:https://test@api.example.com",
				"M_URL":  "https://api.example.com",
				"M_USER": "test",
				"M_PASS": "testpass",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := &CredInitProcessor{
				gitBackend: tt.mockBackend,
			}

			result, err := proc.ProcessCredInitSecrets(tt.secretVars)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				for key, expectedValue := range tt.expected {
					if actualValue, exists := result[key]; !exists {
						t.Errorf("Expected key %s not found in result", key)
					} else if actualValue != expectedValue {
						t.Errorf("For key %s, expected %s but got %s", key, expectedValue, actualValue)
					}
				}

				for key := range result {
					if _, expected := tt.expected[key]; !expected {
						t.Errorf("Unexpected key %s found in result", key)
					}
				}
			}
		})
	}
}
