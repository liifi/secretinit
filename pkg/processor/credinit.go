package processor

import (
	"fmt"

	"github.com/liifi/secretinit/pkg/backend"
	"github.com/liifi/secretinit/pkg/parser"
)

// CredInitProcessor handles credinit-specific credential processing logic
type CredInitProcessor struct {
	gitBackend backend.Backend
}

// NewCredInitProcessor creates a new processor specifically for credinit
func NewCredInitProcessor() *CredInitProcessor {
	return &CredInitProcessor{
		gitBackend: &backend.GitBackend{},
	}
}

// ProcessCredInitSecrets processes secrets with credinit-specific logic:
// - If keyPath is provided, behaves like secretinit (simple replacement)
// - If no keyPath, creates *_URL, *_USER, and *_PASS variables from prefix
func (p *CredInitProcessor) ProcessCredInitSecrets(secretVars map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	for envVar, secretAddress := range secretVars {
		// Parse the secret address
		secretSource, err := parser.ParseSecretString(secretAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secret address for %s: %w", envVar, err)
		}

		// Only process git backend secrets
		if secretSource.Backend != "git" {
			continue
		}

		// If keyPath is specified, behave like secretinit (simple replacement)
		if secretSource.KeyPath != "" {
			value, err := p.gitBackend.RetrieveSecret(secretSource.Service, secretSource.Resource, secretSource.KeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve secret for %s: %w", envVar, err)
			}
			result[envVar] = value
		} else {
			// No keyPath: credinit multi-credential mode
			// Keep original variable unchanged and create additional _URL, _USER, _PASS variables
			// Use the exact variable name as prefix
			prefix := envVar

			// Keep the original variable with its secretinit: value
			result[envVar] = "secretinit:" + secretAddress

			// Retrieve both username and password
			username, err := p.gitBackend.RetrieveSecret(secretSource.Service, secretSource.Resource, "username")
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve username for %s: %w", envVar, err)
			}

			password, err := p.gitBackend.RetrieveSecret(secretSource.Service, secretSource.Resource, "password")
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve password for %s: %w", envVar, err)
			}

			// Create the additional environment variables
			// *_URL gets the clean parsed URL (without username)
			cleanURL, _ := backend.ParseURLForUser(secretSource.Resource)
			result[prefix+"_URL"] = cleanURL
			result[prefix+"_USER"] = username
			result[prefix+"_PASS"] = password
		}
	}

	return result, nil
}
