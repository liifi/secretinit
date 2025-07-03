package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// GCPBackend implements the Backend interface for Google Cloud Platform services.
type GCPBackend struct {
	client *secretmanager.Client
}

// NewGCPBackend creates a new GCPBackend using default GCP credentials.
// This uses the standard GCP SDK credential discovery mechanism (service account, gcloud auth, etc.).
func NewGCPBackend() (*GCPBackend, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Secret Manager client: %w", err)
	}

	return &GCPBackend{
		client: client,
	}, nil
}

// RetrieveSecret retrieves a secret from GCP services.
// The service parameter specifies which GCP service to use: "sm" for Secret Manager.
// The resource format depends on the service:
// - For Secret Manager: "projects/PROJECT_ID/secrets/SECRET_NAME/versions/VERSION" or "PROJECT_ID/SECRET_NAME" or "SECRET_NAME" (uses default project)
// The keyPath is optional and used for JSON key extraction from the secret value.
func (b *GCPBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	switch service {
	case "sm":
		return b.retrieveFromSecretManager(resource, keyPath)
	default:
		return "", fmt.Errorf("unsupported GCP service '%s'. Supported services: 'sm' (Secret Manager)", service)
	}
}

// retrieveFromSecretManager retrieves a secret from GCP Secret Manager.
func (b *GCPBackend) retrieveFromSecretManager(resource, keyPath string) (string, error) {
	// Normalize the resource name to full path format
	secretName := b.normalizeSecretName(resource)

	// Create cache key without keyPath
	cacheKey := fmt.Sprintf("gcp:sm:%s", secretName)

	// Check cache first
	cache := GetGlobalCache()
	if cached, exists := cache.Get(cacheKey); exists {
		// Parse keyPath from cached raw secret value
		if keyPath == "" {
			return cached, nil
		}
		return extractJSONKey(cached, keyPath)
	}

	// Cache miss - retrieve from GCP Secret Manager
	ctx := context.Background()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := b.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from GCP Secret Manager for resource '%s': %w", resource, err)
	}

	if result.Payload == nil || result.Payload.Data == nil {
		return "", fmt.Errorf("no secret value found for resource '%s'", resource)
	}

	// Store raw secret value in cache
	secretValue := string(result.Payload.Data)
	cache.Set(cacheKey, secretValue)

	// Parse keyPath from the raw secret value
	if keyPath == "" {
		return secretValue, nil
	}

	return extractJSONKey(secretValue, keyPath)
}

// normalizeSecretName converts various resource formats to the full GCP Secret Manager resource name.
// Supports:
// - Full path: "projects/PROJECT_ID/secrets/SECRET_NAME/versions/VERSION"
// - Project/secret: "PROJECT_ID/SECRET_NAME" (uses latest version)
// - Secret only: "SECRET_NAME" (uses default project and latest version)
func (b *GCPBackend) normalizeSecretName(resource string) string {
	// If already a full path, return as-is
	if strings.HasPrefix(resource, "projects/") {
		return resource
	}

	// Handle PROJECT_ID/SECRET_NAME format
	if strings.Contains(resource, "/") && !strings.Contains(resource, "projects/") {
		parts := strings.SplitN(resource, "/", 2)
		if len(parts) == 2 {
			projectID := parts[0]
			secretName := parts[1]
			return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
		}
	}

	// Handle SECRET_NAME only - requires GOOGLE_CLOUD_PROJECT env var
	projectID := getGCPProjectID()
	if projectID == "" {
		// Return as-is and let GCP SDK handle the error
		return resource
	}

	return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, resource)
}

// getGCPProjectID attempts to get the GCP project ID from environment variables or metadata.
func getGCPProjectID() string {
	// Try common environment variables
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GCP_PROJECT"); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv("GCLOUD_PROJECT"); projectID != "" {
		return projectID
	}

	// Could also try to get from metadata service, but that's more complex
	// and environment variables are the standard approach
	return ""
}

// Close closes the GCP client connection.
func (b *GCPBackend) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}
