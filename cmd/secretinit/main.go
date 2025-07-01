package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/liifi/secretinit/pkg/backend"
	"github.com/liifi/secretinit/pkg/env"
	"github.com/liifi/secretinit/pkg/mappings"
	"github.com/liifi/secretinit/pkg/processor"
)

func main() {
	// Parse mappings and command arguments
	mappingMap, cmdStart := mappings.ParseMappingsFromArgs(os.Args)

	if cmdStart >= len(os.Args) {
		fmt.Fprintf(os.Stderr, "Usage: secretinit [--mappings|-m SOURCE->TARGET,SOURCE2->TARGET2] <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  secretinit myapp --config /etc/myapp/config.yaml\n")
		fmt.Fprintf(os.Stderr, "  secretinit -m \"DB_USER->DATABASE_USERNAME,DB_PASS->DATABASE_PASSWORD\" myapp\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment variables with 'secretinit:' prefix will be resolved:\n")
		fmt.Fprintf(os.Stderr, "  export DB_PASSWORD=\"secretinit:git:https://github.com/myorg/secrets.git:::password\"\n")
		fmt.Fprintf(os.Stderr, "  export API_KEY=\"secretinit:aws:sm:myapp/api-key\"\n")
		os.Exit(1)
	}

	// Scan environment variables for the secretinit: prefix
	secretEnvVars := env.ScanSecretEnvVars()

	// Create processor and register backends
	proc := processor.NewSecretProcessor()
	proc.RegisterBackend("git", &backend.GitBackend{})

	// Register AWS backend
	awsBackend, err := backend.NewAWSBackend()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize AWS backend: %v\n", err)
		fmt.Fprintf(os.Stderr, "AWS Secrets Manager functionality will not be available.\n")
	} else {
		proc.RegisterBackend("aws", awsBackend)
	}

	// TODO: Register other backends as they are implemented
	// proc.RegisterBackend("gcp", &backend.GCPBackend{})
	// proc.RegisterBackend("azure", &backend.AzureBackend{})

	// Process secrets
	retrievedSecrets, err := proc.ProcessSecrets(secretEnvVars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing secrets: %v\n", err)
		os.Exit(1)
	}

	// Apply mappings
	finalEnv, err := mappings.ApplyMappings(retrievedSecrets, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error applying mappings: %v\n", err)
		os.Exit(1)
	}

	// Prepare the environment for the new process
	newEnv := os.Environ() // Start with the current environment

	// Add resolved secrets to environment
	for key, value := range finalEnv {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Apply command-line mappings
	newEnv = mappings.ApplyMappingsToEnv(newEnv, mappingMap)

	// Execute the command
	cmd := exec.Command(os.Args[cmdStart], os.Args[cmdStart+1:]...)
	cmd.Env = newEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
