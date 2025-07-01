package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/liifi/secretinit/pkg/backend"
	"github.com/liifi/secretinit/pkg/env"
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
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("credinit version %s\n", version)
		return
	}

	if len(os.Args) < 2 {
		binaryName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [--store --url URL --user USER] [--mappings|-m SOURCE->TARGET,SOURCE2->TARGET2] <command> [args..]\n", binaryName)
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --store --url https://api.example.com --user myuser\n", binaryName)
		fmt.Fprintf(os.Stderr, "  MYAPP=secretinit:git:https://api.example.com %s myapp arg1\n", binaryName)
		fmt.Fprintf(os.Stderr, "  TOKEN=secretinit:git:https://api.example.com:::password %s myapp arg1\n", binaryName)
		fmt.Fprintf(os.Stderr, "  %s -m \"MYAPP_USER->DB_USERNAME,MYAPP_PASS->DB_PASSWORD\" myapp arg1\n", binaryName)
		fmt.Fprintf(os.Stderr, "  CREDINIT_LOG_LEVEL=DEBUG %s myapp arg1\n", binaryName)
		fmt.Fprintf(os.Stderr, "\nRequirements:\n")
		fmt.Fprintf(os.Stderr, "  - Git must be installed (credential retrieval will silently skip if missing)\n")
		fmt.Fprintf(os.Stderr, "  - Configure git credential helper for secure storage\n")
		os.Exit(1)
	}

	// Handle credential storage
	if os.Args[1] == "--store" {
		handleStore()
		return
	}

	// Parse mappings and find command start
	mappingMap, cmdStart := mappings.ParseMappingsFromArgs(os.Args)
	debugLog("Parsed mappings: %+v, command starts at arg %d", mappingMap, cmdStart)

	// Load credential and execute command
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

	// Apply mappings to environment
	currentEnv := os.Environ()

	// Add resolved secrets to current environment
	for key, value := range retrievedSecrets {
		currentEnv = append(currentEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Apply any specified mappings
	finalEnv := mappings.ApplyMappingsToEnv(currentEnv, mappingMap)

	debugLog("Executing command: %v", os.Args[cmdStart:])
	executeCommand(os.Args[cmdStart:], finalEnv)
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

	// Prompt for URL if not provided
	if url == "" {
		fmt.Print("URL: ")
		if _, err := fmt.Scanln(&url); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading URL: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse the URL to extract user if present
	parsedURL, userFromURL := backend.ParseURLForUser(url)

	// If user was not provided via flag, use the one from the URL if available
	if user == "" {
		user = userFromURL
	}

	// Prompt for Username if still not provided
	if user == "" {
		fmt.Print("Username: ")
		if _, err := fmt.Scanln(&user); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading username: %v\n", err)
			os.Exit(1)
		}
	}

	// Build input for git credential reject (to clear any existing credentials)
	input := fmt.Sprintf("url=%s\n", parsedURL)
	if user != "" {
		input += fmt.Sprintf("username=%s\n", user)
	}
	input += "\n" // Important: git credential reject/fill expects a blank line to terminate input

	debugLog("Calling git credential reject:\n%s", input)
	cmd := exec.Command("git", "credential", "reject")
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr // Allow git to show prompts/errors
	cmd.Run()              // Ignore errors - credential might not exist

	// Now prompt for credentials (git credential fill will ask for password if not provided)
	// Use parsedURL and the determined user for git credential commands
	input = fmt.Sprintf("url=%s\n", parsedURL)
	if user != "" {
		input += fmt.Sprintf("username=%s\n", user)
	}
	input += "\n"

	debugLog("Calling git credential fill:\n%s", input)
	cmd = exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr // Allow git to show prompts/errors
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get credentials: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you have a git credential helper configured \n")
		os.Exit(1)
	}

	debugLog("Calling git credential approve:\n%+v", input)
	// Now use git credential approve to store the credentials
	// input = fmt.Sprintf("url=%s\n", parsedURL) // Use parsedURL here
	cmd = exec.Command("git", "credential", "approve")
	cmd.Stdin = strings.NewReader(string(output))
	cmd.Stderr = os.Stderr // Allow git to show prompts/errors
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to store credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Credentials stored successfully")
}

// executeCommand executes the given command with the provided environment variables.
func executeCommand(args []string, env []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided to execute.")
		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
		os.Exit(1)
	}
	debugLog("Started process with PID: %d", cmd.Process.Pid)

	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to the child process
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}
