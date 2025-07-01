package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	cfg, err := config.LoadDefaultConfig(context.TODO())
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
	switch service {
	case "sm":
		return b.retrieveFromSecretsManager(resource, keyPath)
	case "ps":
		return b.retrieveFromParameterStore(resource, keyPath)
	default:
		return "", fmt.Errorf("unsupported AWS service '%s'. Supported services: 'sm' (Secrets Manager), 'ps' (Parameter Store)", service)
	}
}

// retrieveFromSecretsManager retrieves a secret from AWS Secrets Manager.
func (b *AWSBackend) retrieveFromSecretsManager(resource, keyPath string) (string, error) {
	ctx := context.TODO()

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

	// If no keyPath is specified, return the raw secret value
	if keyPath == "" {
		return secretValue, nil
	}

	// Try to parse as JSON and extract the specified key
	return extractJSONKey(secretValue, keyPath)
}

// retrieveFromParameterStore retrieves a parameter from AWS Systems Manager Parameter Store.
func (b *AWSBackend) retrieveFromParameterStore(resource, keyPath string) (string, error) {
	ctx := context.TODO()

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

	// If no keyPath is specified, return the raw parameter value
	if keyPath == "" {
		return paramValue, nil
	}

	// Try to parse as JSON and extract the specified key
	return extractJSONKey(paramValue, keyPath)
}

// extractJSONKey attempts to parse the secret value as JSON and extract the specified key.
func extractJSONKey(secretValue, keyPath string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(secretValue), &data); err != nil {
		return "", fmt.Errorf("failed to parse secret value as JSON for key extraction '%s': %w", keyPath, err)
	}

	// Support nested key paths using dot notation (e.g., "database.password")
	keys := strings.Split(keyPath, ".")
	var current interface{} = data

	for i, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[key]
			if !exists {
				return "", fmt.Errorf("key '%s' not found in secret JSON (at path segment %d: '%s')", keyPath, i, key)
			}
			current = val
		default:
			return "", fmt.Errorf("cannot navigate to key '%s': intermediate value at segment %d ('%s') is not a JSON object", keyPath, i, key)
		}
	}

	// Convert the final value to string
	switch v := current.(type) {
	case string:
		return v, nil
	case nil:
		return "", fmt.Errorf("key '%s' has null value in secret JSON", keyPath)
	default:
		// For non-string values, convert to JSON string representation
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to convert key '%s' value to string: %w", keyPath, err)
		}
		return string(jsonBytes), nil
	}
}
