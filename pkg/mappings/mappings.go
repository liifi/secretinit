package mappings

import (
	"fmt"
	"os"
	"strings"
)

// ApplyMappings takes a map of environment variables and a mapping string
// and applies the mappings to the environment map.
// The mapping string should be in the format "TARGET=SOURCE,TARGET2=SOURCE2".
func ApplyMappings(env map[string]string, mappings string) (map[string]string, error) {
	if mappings == "" {
		return env, nil
	}

	mappingPairs := strings.Split(mappings, ",")
	appliedEnv := make(map[string]string)

	// Copy original environment variables
	for key, value := range env {
		appliedEnv[key] = value
	}

	for _, pair := range mappingPairs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format: %s", pair)
		}
		target := strings.TrimSpace(parts[0])
		source := strings.TrimSpace(parts[1])
		// Apply mapping: if source exists, set target to source's value
		if value, ok := appliedEnv[source]; ok {
			appliedEnv[target] = value
		}
	}
	return appliedEnv, nil
}

// ParseMappingsFromArgs parses --mappings or -m flags from command line arguments
// and also checks the SECRETINIT_MAPPINGS environment variable.
// Returns the parsed mappings map and the index where the actual command starts
func ParseMappingsFromArgs(args []string) (map[string]string, int) {
	mappings := make(map[string]string)
	cmdStart := 1 // Default: command starts after the binary name

	// First, check for SECRETINIT_MAPPINGS environment variable
	if envMappings := os.Getenv("SECRETINIT_MAPPINGS"); envMappings != "" {
		ParseMappingString(envMappings, mappings)
	}

	// Then parse command line arguments (these override environment variable)
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--mappings" || arg == "-m" {
			if i+1 < len(args) {
				ParseMappingString(args[i+1], mappings)
				i++              // Skip the mapping value
				cmdStart = i + 1 // Command starts after this
			}
		} else {
			// First non-mapping argument is the start of the command
			cmdStart = i
			break
		}
	}

	return mappings, cmdStart
}

// ParseMappingString parses a comma-separated string of TARGET=SOURCE mappings
func ParseMappingString(mappingStr string, mappings map[string]string) {
	if mappingStr == "" {
		return
	}

	pairs := strings.Split(mappingStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			target := strings.TrimSpace(parts[0])
			source := strings.TrimSpace(parts[1])
			mappings[target] = source
		}
	}
}

// ApplyMappingsToEnv applies mappings to a slice of environment variables (KEY=VALUE format)
func ApplyMappingsToEnv(env []string, mappings map[string]string) []string {
	if len(mappings) == 0 {
		return env
	}

	envMap := make(map[string]string)

	// Convert slice to map
	for _, envVar := range env {
		if parts := strings.SplitN(envVar, "=", 2); len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Apply mappings
	for target, source := range mappings {
		if value, exists := envMap[source]; exists {
			envMap[target] = value
		}
	}

	// Convert back to slice
	result := make([]string, 0, len(envMap))
	for key, value := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}
