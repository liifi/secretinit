package backend

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWSBackend implements the Backend interface for AWS services (Secrets Manager and Parameter Store).
type AWSBackend struct {
	secretsClient *secretsmanager.Client
	ssmClient     *ssm.Client
}

// NewAWSBackend creates a new AWSBackend using default AWS SDK configuration.
// This uses the standard AWS SDK credential and region discovery mechanism.
func NewAWSBackend() (*AWSBackend, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	secretsClient := secretsmanager.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	return &AWSBackend{
		secretsClient: secretsClient,
		ssmClient:     ssmClient,
	}, nil
}

// RetrieveSecret retrieves a secret from AWS services (Secrets Manager or Parameter Store).
// The service parameter specifies which AWS service to use: "sm" for Secrets Manager, "ps" for Parameter Store.
// The resource can be either a simple name or a full ARN for Secrets Manager, or parameter name/path for Parameter Store.
// The keyPath is optional and used for JSON key extraction from the secret value.
func (b *AWSBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	cache := GetGlobalCache()

	// Create cache key for the raw secret (without keyPath since that's just parsing)
	cacheKey := fmt.Sprintf("aws:%s:%s", service, resource)

	// Check if we have cached the raw secret value
	var rawSecretValue string
	if cached, exists := cache.Get(cacheKey); exists {
		rawSecretValue = cached
	} else {
		// Cache miss - retrieve from AWS
		var err error
		switch service {
		case "sm":
			rawSecretValue, err = b.retrieveFromSecretsManager(resource)
		case "ps":
			rawSecretValue, err = b.retrieveFromParameterStore(resource)
		default:
			return "", fmt.Errorf("unsupported AWS service '%s'. Supported services: 'sm' (Secrets Manager), 'ps' (Parameter Store)", service)
		}

		if err != nil {
			return "", err
		}

		// Cache the raw secret value
		cache.Set(cacheKey, rawSecretValue)
	}

	// Apply keyPath parsing to the raw value
	if keyPath == "" {
		return rawSecretValue, nil
	}

	// Try to parse as JSON and extract the specified key
	return extractJSONKey(rawSecretValue, keyPath)
}

// retrieveFromSecretsManager retrieves a secret from AWS Secrets Manager.
func (b *AWSBackend) retrieveFromSecretsManager(resource string) (string, error) {
	ctx := context.Background()

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &resource,
	}

	result, err := b.secretsClient.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from AWS Secrets Manager for resource '%s': %w", resource, err)
	}

	// AWS Secrets Manager can return either SecretString or SecretBinary
	var secretValue string
	if result.SecretString != nil {
		secretValue = *result.SecretString
	} else if result.SecretBinary != nil {
		secretValue = string(result.SecretBinary)
	} else {
		return "", fmt.Errorf("no secret value found for resource '%s'", resource)
	}

	return secretValue, nil
}

// retrieveFromParameterStore retrieves a parameter from AWS Systems Manager Parameter Store.
func (b *AWSBackend) retrieveFromParameterStore(resource string) (string, error) {
	ctx := context.Background()

	input := &ssm.GetParameterInput{
		Name:           &resource,
		WithDecryption: &[]bool{true}[0], // Always decrypt SecureString parameters
	}

	result, err := b.ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve parameter from AWS Parameter Store for resource '%s': %w", resource, err)
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		return "", fmt.Errorf("no parameter value found for resource '%s'", resource)
	}

	paramValue := *result.Parameter.Value
	return paramValue, nil
}
