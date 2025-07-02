package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file
// Returns a map of key-value pairs, or an error if the file cannot be read
func LoadEnvFile(filepath string) (map[string]string, error) {
	envVars := make(map[string]string)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d in %s: %s", lineNum, filepath, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty key on line %d in %s", lineNum, filepath)
		}

		envVars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %v", filepath, err)
	}

	return envVars, nil
}

// LoadAndSetEnvFile loads a .env file and sets the variables in the current process
// Returns the number of variables loaded, or an error
func LoadAndSetEnvFile(filepath string) (int, error) {
	envVars, err := LoadEnvFile(filepath)
	if err != nil {
		return 0, err
	}

	count := 0
	for key, value := range envVars {
		// Only set if not already set (system env vars take precedence)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			count++
		}
	}

	return count, nil
}

// LoadAndSetEnvFileOverride loads a .env file and sets the variables in the current process
// .env file variables override existing environment variables
func LoadAndSetEnvFileOverride(filepath string) (int, error) {
	envVars, err := LoadEnvFile(filepath)
	if err != nil {
		return 0, err
	}

	count := 0
	for key, value := range envVars {
		os.Setenv(key, value)
		count++
	}

	return count, nil
}
