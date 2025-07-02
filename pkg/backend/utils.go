package backend

import (
	"encoding/json"
	"fmt"
	"strings"
)

// extractJSONKey attempts to parse the secret value as JSON and extract the specified key.
// This is a shared utility function used by multiple backends for JSON key extraction.
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
