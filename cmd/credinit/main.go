package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/liifi/secretinit/pkg/backend"
	"github.com/liifi/secretinit/pkg/env"
	executil "github.com/liifi/secretinit/pkg/exec"
	"github.com/liifi/secretinit/pkg/mappings"
	"github.com/liifi/secretinit/pkg/processor"
)

// Version information set by GoReleaser
var ( //goreleaser
	version = "dev"
)

var debugEnabled = os.Getenv("CREDINIT_LOG_LEVEL") == "DEBUG"

// debugLog prints debug messages to stderr if debugEnabled is true.
func debugLog(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func main() {
	binaryName := filepath.Base(os.Args[0])

	// Handle help and version flags first
	if len(os.Args) <= 1 {
		showHelp(binaryName)
		os.Exit(1)
	}

	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			showHelp(binaryName)
			return
		}
		if arg == "-v" || arg == "--version" {
			fmt.Printf("%s version %s\n", binaryName, version)
			return
		}
	}

	// Parse command line arguments for -o/--stdout flag
	var stdout bool
	var secretAddress string

	// Parse flags
	args := os.Args[1:]
	filteredArgs := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--stdout":
			stdout = true
			if i+1 < len(args) {
				secretAddress = args[i+1]
				i++ // Skip the next argument as it's the secret address
			} else {
				fmt.Fprintf(os.Stderr, "Error: -o/--stdout requires a secret address argument\n")
				os.Exit(1)
			}
		default:
			filteredArgs = append(filteredArgs, args[i])
		}
	}

	if len(filteredArgs) < 1 && !stdout {
		showHelp(binaryName)
		os.Exit(1)
	}

	// Handle credential storage
	if len(filteredArgs) > 0 && filteredArgs[0] == "--store" {
		handleStore()
		return
	}

	// Parse mappings and command arguments from filtered args
	mappingMap, cmdStart := mappings.ParseMappingsFromArgs(append([]string{os.Args[0]}, filteredArgs...))

	// Adjust cmdStart since we removed the program name
	if cmdStart > 0 {
		cmdStart--
	}

	debugLog("Parsed mappings: %+v, command starts at arg %d", mappingMap, cmdStart)

	// Handle -o/--stdout flag
	if stdout {
		value, err := processor.ProcessSingleCredInitSecret(secretAddress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(value)
		return
	}

	// Scan environment variables for the secretinit: prefix
	secretEnvVars := env.ScanSecretEnvVars()

	// Filter for git backend only (credinit is git-specific)
	gitSecrets := processor.FilterByBackend(secretEnvVars, "git")
	debugLog("Found %d git secrets to process", len(gitSecrets))

	// Create credinit-specific processor
	credInitProc := processor.NewCredInitProcessor()

	// Process secrets with credinit logic
	retrievedSecrets, err := credInitProc.ProcessCredInitSecrets(gitSecrets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing secrets: %v\n", err)
		os.Exit(1)
	}

	// Prepare the environment for the new process
	newEnv := os.Environ() // Start with the current environment

	// Add resolved secrets to environment
	for key, value := range retrievedSecrets {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Apply any specified mappings
	finalEnv := mappings.ApplyMappingsToEnv(newEnv, mappingMap)

	debugLog("Executing command: %v", filteredArgs[cmdStart:])
	executil.ExecuteCommandWithDebug(filteredArgs[cmdStart:], finalEnv, debugLog)
}

// handleStore manages the storage of credentials using git credential helper.
func handleStore() {
	var url, user string

	for i, arg := range os.Args {
		if arg == "--url" && i+1 < len(os.Args) {
			url = os.Args[i+1]
		}
		if arg == "--user" && i+1 < len(os.Args) {
			user = os.Args[i+1]
		}
	}

	// Use git backend to store credentials (will prompt for URL if empty)
	gitBackend := &backend.GitBackend{}
	if err := gitBackend.StoreCredential(url, user); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to store credentials: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you have a git credential helper configured\n")
		os.Exit(1)
	}

	fmt.Println("Credentials stored successfully")
}

// showHelp displays the help message for credinit
func showHelp(binaryName string) {
	fmt.Fprintf(os.Stderr, "Usage: %s [-h|--help] [-v|--version] [-o|--stdout SECRET_ADDRESS] [--store --url URL --user USER] [--mappings|-m SOURCE->TARGET,SOURCE2->TARGET2] <command> [args..]\n", binaryName)
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	fmt.Fprintf(os.Stderr, "  -h, --help              Show this help message\n")
	fmt.Fprintf(os.Stderr, "  -v, --version           Show version information\n")
	fmt.Fprintf(os.Stderr, "  -o, --stdout ADDRESS    Output a single secret to stdout\n")
	fmt.Fprintf(os.Stderr, "  --store                 Store credentials using git credential helper\n")
	fmt.Fprintf(os.Stderr, "  --url URL               URL for credential storage\n")
	fmt.Fprintf(os.Stderr, "  --user USER             Username for credential storage\n")
	fmt.Fprintf(os.Stderr, "  -m, --mappings MAP      Environment variable mappings\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s --store --url https://api.example.com --user myuser\n", binaryName)
	fmt.Fprintf(os.Stderr, "  MYAPP=secretinit:git:https://api.example.com %s myapp arg1\n", binaryName)
	fmt.Fprintf(os.Stderr, "  TOKEN=secretinit:git:https://api.example.com:::password %s myapp arg1\n", binaryName)
	fmt.Fprintf(os.Stderr, "  %s -m \"MYAPP_USER->DB_USERNAME,MYAPP_PASS->DB_PASSWORD\" myapp arg1\n", binaryName)
	fmt.Fprintf(os.Stderr, "  %s -o \"git:https://api.example.com:::password\"\n", binaryName)
	fmt.Fprintf(os.Stderr, "  %s --stdout \"git:https://api.example.com:::username\"\n", binaryName)
	fmt.Fprintf(os.Stderr, "  CREDINIT_LOG_LEVEL=DEBUG %s myapp arg1\n", binaryName)
	fmt.Fprintf(os.Stderr, "\nNote: The 'secretinit:' prefix is automatically added if not present.\n")
	fmt.Fprintf(os.Stderr, "Requirements:\n")
	fmt.Fprintf(os.Stderr, "  - Git must be installed (credential retrieval will silently skip if missing)\n")
	fmt.Fprintf(os.Stderr, "  - Configure git credential helper for secure storage\n")
}
