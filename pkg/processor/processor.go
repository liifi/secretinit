package processor

import (
	"fmt"
	"strings"

	"github.com/liifi/secretinit/pkg/backend"
	"github.com/liifi/secretinit/pkg/parser"
)

// SecretProcessor handles the processing of secret environment variables
type SecretProcessor struct {
	backends map[string]backend.Backend
}

// NewSecretProcessor creates a new SecretProcessor with the given backends
func NewSecretProcessor() *SecretProcessor {
	return &SecretProcessor{
		backends: make(map[string]backend.Backend),
	}
}

// RegisterBackend registers a backend for a specific backend type
func (p *SecretProcessor) RegisterBackend(backendType string, b backend.Backend) {
	p.backends[backendType] = b
}

// ProcessSecrets processes a map of secret environment variables and returns resolved values
func (p *SecretProcessor) ProcessSecrets(secretVars map[string]string) (map[string]string, error) {
	resolvedSecrets := make(map[string]string)

	for varName, secretAddress := range secretVars {
		// Parse the secret address using the parser package
		secretSource, err := parser.ParseSecretString(secretAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secret address for variable '%s': %w", varName, err)
		}

		// Check if we have a backend registered for this backend type
		backend, exists := p.backends[secretSource.Backend]
		if !exists {
			return nil, fmt.Errorf("unsupported backend '%s' for variable '%s'", secretSource.Backend, varName)
		}

		// Validate service field for specific backends
		if secretSource.Backend == "aws" && secretSource.Service != "sm" && secretSource.Service != "ps" {
			return nil, fmt.Errorf("unsupported AWS service '%s' for variable '%s'. Supported services: 'sm' (Secrets Manager), 'ps' (Parameter Store)", secretSource.Service, varName)
		}

		// Determine the keyPath - use "password" as default for git if not specified
		keyPath := secretSource.KeyPath
		if secretSource.Backend == "git" && keyPath == "" {
			keyPath = "password"
		}

		// Retrieve the secret value from the backend
		secretValue, err := backend.RetrieveSecret(secretSource.Service, secretSource.Resource, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve secret for variable '%s' (%s): %w", varName, secretAddress, err)
		}

		resolvedSecrets[varName] = secretValue
	}

	return resolvedSecrets, nil
}

// FilterByBackend filters secret variables to only include those for a specific backend
func FilterByBackend(secretVars map[string]string, backendType string) map[string]string {
	filtered := make(map[string]string)
	prefix := backendType + ":"

	for varName, secretAddress := range secretVars {
		if strings.HasPrefix(secretAddress, prefix) {
			filtered[varName] = secretAddress
		}
	}

	return filtered
}
