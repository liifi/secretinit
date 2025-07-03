package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// AzureBackend implements the Backend interface for Azure services.
type AzureBackend struct {
	keyVaultClients map[string]*azsecrets.Client
}

// NewAzureBackend creates a new AzureBackend using default Azure SDK configuration.
// This uses the standard Azure SDK credential chain (environment variables,
// managed identity, Azure CLI, etc.).
func NewAzureBackend() (*AzureBackend, error) {
	return &AzureBackend{
		keyVaultClients: make(map[string]*azsecrets.Client),
	}, nil
}

// RetrieveSecret retrieves a secret from Azure services.
// The service parameter specifies which Azure service to use: "kv" for Key Vault.
// The resource should be in the format "vault-name/secret-name" or "vault-name/secret-name/version".
// The keyPath is optional and used for JSON key extraction from the secret value.
func (b *AzureBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	switch service {
	case "kv":
		return b.retrieveFromKeyVault(resource, keyPath)
	default:
		return "", fmt.Errorf("unsupported Azure service '%s'. Supported services: 'kv' (Key Vault)", service)
	}
}

// retrieveFromKeyVault retrieves a secret from Azure Key Vault.
func (b *AzureBackend) retrieveFromKeyVault(resource, keyPath string) (string, error) {
	// Parse the resource to extract vault name, secret name, and optional version
	vaultName, secretName, version, err := b.parseKeyVaultResource(resource)
	if err != nil {
		return "", fmt.Errorf("failed to parse Key Vault resource '%s': %w", resource, err)
	}

	// Create cache key without keyPath (include version if specified)
	var cacheKey string
	if version != "" {
		cacheKey = fmt.Sprintf("azure:kv:%s/%s/%s", vaultName, secretName, version)
	} else {
		cacheKey = fmt.Sprintf("azure:kv:%s/%s", vaultName, secretName)
	}

	// Check cache first
	cache := GetGlobalCache()
	if cached, exists := cache.Get(cacheKey); exists {
		// Parse keyPath from cached raw secret value
		if keyPath == "" {
			return cached, nil
		}
		return extractJSONKey(cached, keyPath)
	}

	// Cache miss - retrieve from Azure Key Vault
	ctx := context.Background()

	// Get or create client for this vault
	client, err := b.getKeyVaultClient(vaultName)
	if err != nil {
		return "", fmt.Errorf("failed to create Key Vault client for vault '%s': %w", vaultName, err)
	}

	// Retrieve the secret
	var response azsecrets.GetSecretResponse
	if version != "" {
		response, err = client.GetSecret(ctx, secretName, version, nil)
	} else {
		response, err = client.GetSecret(ctx, secretName, "", nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret '%s' from Azure Key Vault '%s': %w", secretName, vaultName, err)
	}

	if response.Value == nil {
		return "", fmt.Errorf("no secret value found for '%s' in vault '%s'", secretName, vaultName)
	}

	// Store raw secret value in cache
	secretValue := *response.Value
	cache.Set(cacheKey, secretValue)

	// Parse keyPath from the raw secret value
	if keyPath == "" {
		return secretValue, nil
	}

	return extractJSONKey(secretValue, keyPath)
}

// parseKeyVaultResource parses the resource string into vault name, secret name, and optional version.
// Supports formats:
// - "vault-name/secret-name" (latest version)
// - "vault-name/secret-name/version"
func (b *AzureBackend) parseKeyVaultResource(resource string) (vaultName, secretName, version string, err error) {
	parts := strings.Split(resource, "/")

	switch len(parts) {
	case 2:
		// Format: vault-name/secret-name (latest version)
		return parts[0], parts[1], "", nil
	case 3:
		// Format: vault-name/secret-name/version
		return parts[0], parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("invalid Key Vault resource format: %s. Expected 'vault-name/secret-name' or 'vault-name/secret-name/version'", resource)
	}
}

// getKeyVaultClient gets or creates a Key Vault client for the specified vault.
func (b *AzureBackend) getKeyVaultClient(vaultName string) (*azsecrets.Client, error) {
	// Check if we already have a client for this vault
	if client, exists := b.keyVaultClients[vaultName]; exists {
		return client, nil
	}

	// Create credential using default credential chain
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	// Construct the Key Vault URL
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)

	// Create the Key Vault client
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client for vault '%s': %w", vaultName, err)
	}

	// Cache the client for future use
	b.keyVaultClients[vaultName] = client

	return client, nil
}

// Close performs cleanup for the Azure backend.
func (b *AzureBackend) Close() error {
	// Azure SDK clients don't require explicit cleanup, but we can clear the cache
	b.keyVaultClients = make(map[string]*azsecrets.Client)
	return nil
}
