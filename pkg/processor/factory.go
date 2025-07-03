package processor

import (
	"fmt"
	"strings"
)

// NewProcessorForSecrets creates a processor with only the backends needed for the given secrets
func NewProcessorForSecrets(secrets map[string]string) (*SecretProcessor, error) {
	// Scan secrets to determine which backends are needed
	neededBackends := ScanForRequiredBackends(secrets)

	return NewProcessorWithBackends(neededBackends)
}

// NewProcessorWithBackends creates a processor with the specified backends
func NewProcessorWithBackends(backendNames []string) (*SecretProcessor, error) {
	proc := NewSecretProcessor()

	// Get available backends based on build tags
	backendFactories := RegisterAllBackends()

	for _, name := range backendNames {
		factory, exists := backendFactories[name]
		if !exists {
			return nil, fmt.Errorf("backend not available in this build: %s", name)
		}

		backend, err := factory()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize %s backend: %v", name, err)
		}

		proc.RegisterBackend(name, backend)
	}

	return proc, nil
}

// ScanForRequiredBackends scans secrets to determine which backends are needed
func ScanForRequiredBackends(secrets map[string]string) []string {
	backendSet := make(map[string]bool)

	for _, secretAddr := range secrets {
		var backendPart string

		if strings.HasPrefix(secretAddr, "secretinit:") {
			// Handle prefixed format: secretinit:git:...
			parts := strings.Split(secretAddr, ":")
			if len(parts) >= 2 {
				backendPart = parts[1]
			}
		} else {
			// Handle direct format: git:...
			parts := strings.Split(secretAddr, ":")
			if len(parts) >= 1 {
				backendPart = parts[0]
			}
		}

		if backendPart != "" {
			backendSet[backendPart] = true
		}
	}

	var backends []string
	for backend := range backendSet {
		backends = append(backends, backend)
	}
	return backends
}

// ProcessSingleSecret is a convenience function for processing a single secret
func ProcessSingleSecret(secretAddress string) (string, error) {
	// Remove secretinit: prefix if present, as the processor expects raw backend format
	secretAddress = strings.TrimPrefix(secretAddress, "secretinit:")

	secrets := map[string]string{"TEMP_KEY": secretAddress}
	proc, err := NewProcessorForSecrets(secrets)
	if err != nil {
		return "", err
	}

	retrievedSecrets, err := proc.ProcessSecrets(secrets)
	if err != nil {
		return "", err
	}

	if value, exists := retrievedSecrets["TEMP_KEY"]; exists {
		return value, nil
	}
	return "", fmt.Errorf("secret not found")
}
