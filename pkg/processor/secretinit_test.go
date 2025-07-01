package processor

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// MockAWSBackend for testing
type MockAWSBackend struct {
	secretValue string
	err         error
}

func (m *MockAWSBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	secretValue := m.secretValue

	// If no keyPath is specified, return the raw secret value
	if keyPath == "" {
		return secretValue, nil
	}

	// Try to parse as JSON and extract the specified key (simplified version)
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(secretValue), &data); err != nil {
		return secretValue, nil // If not JSON, return raw value
	}

	// Support nested key paths using dot notation (e.g., "database.password")
	keys := strings.Split(keyPath, ".")
	var current interface{} = data

	for _, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[key]
			if !exists {
				return "", errors.New("key not found")
			}
			current = val
		default:
			return "", errors.New("cannot navigate to key")
		}
	}

	// Convert the final value to string
	switch v := current.(type) {
	case string:
		return v, nil
	default:
		// For non-string values, convert to JSON string representation
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes), nil
	}
}

func TestSecretProcessor_ProcessSecrets_AWS(t *testing.T) {
	tests := []struct {
		name        string
		secretVars  map[string]string
		mockBackend *MockAWSBackend
		expected    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "AWS Secrets Manager - valid service",
			secretVars: map[string]string{
				"DB_PASSWORD": "aws:sm:myapp/db-creds:::password",
			},
			mockBackend: &MockAWSBackend{
				secretValue: "secret123",
			},
			expected: map[string]string{
				"DB_PASSWORD": "secret123",
			},
			expectError: false,
		},
		{
			name: "AWS - invalid service",
			secretVars: map[string]string{
				"DB_PASSWORD": "aws:invalid:myapp/db-creds:::password",
			},
			mockBackend: &MockAWSBackend{
				secretValue: "secret123",
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "unsupported AWS service 'invalid' for variable 'DB_PASSWORD'. Supported services: 'sm' (Secrets Manager), 'ps' (Parameter Store)",
		},
		{
			name: "AWS Parameter Store - valid service",
			secretVars: map[string]string{
				"APP_CONFIG": "aws:ps:/myapp/config",
			},
			mockBackend: &MockAWSBackend{
				secretValue: "config-value",
			},
			expected: map[string]string{
				"APP_CONFIG": "config-value",
			},
			expectError: false,
		},
		{
			name: "AWS Parameter Store - with keyPath",
			secretVars: map[string]string{
				"DB_HOST": "aws:ps:/myapp/db-config:::host",
			},
			mockBackend: &MockAWSBackend{
				secretValue: `{"host":"db.example.com","port":5432}`,
			},
			expected: map[string]string{
				"DB_HOST": "db.example.com",
			},
			expectError: false,
		},
		{
			name: "AWS Secrets Manager - no keyPath",
			secretVars: map[string]string{
				"DB_CREDS": "aws:sm:myapp/db-creds",
			},
			mockBackend: &MockAWSBackend{
				secretValue: `{"username":"dbuser","password":"dbpass"}`,
			},
			expected: map[string]string{
				"DB_CREDS": `{"username":"dbuser","password":"dbpass"}`,
			},
			expectError: false,
		},
		{
			name: "AWS Secrets Manager - backend error",
			secretVars: map[string]string{
				"API_KEY": "aws:sm:myapp/api-key",
			},
			mockBackend: &MockAWSBackend{
				err: errors.New("secret not found"),
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "failed to retrieve secret for variable 'API_KEY' (aws:sm:myapp/api-key): secret not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := NewSecretProcessor()
			proc.RegisterBackend("aws", tt.mockBackend)

			result, err := proc.ProcessSecrets(tt.secretVars)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d", len(tt.expected), len(result))
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Missing key '%s' in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		})
	}
}
