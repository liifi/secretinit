package processor

import (
	"fmt"

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

// ClearCache clears all caches for all registered backends
func (p *SecretProcessor) ClearCache() {
	backend.ClearGlobalCache()
}

// GetCacheStats returns cache statistics for all backends
func (p *SecretProcessor) GetCacheStats() map[string]int {
	stats := make(map[string]int)
	// Since we're using a global cache, return total cache size for all backends
	totalSize := backend.GetGlobalCacheSize()
	for backendType := range p.backends {
		// We can't easily separate cache sizes by backend type with global cache
		// So we'll show total for each backend type
		stats[backendType] = totalSize
	}
	return stats
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

		// Handle git backend multi-credential expansion when no keyPath is specified
		if secretSource.Backend == "git" && secretSource.KeyPath == "" {
			// Multi-credential mode: create _URL, _USER, _PASS variables
			// Keep original variable unchanged with secretinit: prefix
			resolvedSecrets[varName] = "secretinit:" + secretAddress

			// Retrieve both username and password
			username, err := backend.RetrieveSecret(secretSource.Service, secretSource.Resource, "username")
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve username for variable '%s' (%s): %w", varName, secretAddress, err)
			}

			password, err := backend.RetrieveSecret(secretSource.Service, secretSource.Resource, "password")
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve password for variable '%s' (%s): %w", varName, secretAddress, err)
			}

			// Create the additional environment variables
			// *_URL gets the clean parsed URL (without username)
			cleanURL, _ := parser.ParseGitURL(secretSource.Resource)
			resolvedSecrets[varName+"_URL"] = cleanURL
			resolvedSecrets[varName+"_USER"] = username
			resolvedSecrets[varName+"_PASS"] = password
		} else {
			// Single credential mode (existing logic)
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
	}

	return resolvedSecrets, nil
}
