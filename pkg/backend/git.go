package backend

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/liifi/secretinit/pkg/parser"
)

// GitBackend implements the Backend interface for the Git credential manager.
type GitBackend struct{}

// RetrieveSecret retrieves a secret from the Git credential manager.
// The service parameter is empty for git (git doesn't have services).
// The resource string may contain username (e.g., "https://user@example.com").
// The keyPath should be "username" or "password".
func (b *GitBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	cache := GetGlobalCache()
	// Create cache key for the credential (without keyPath since we cache the full credential)
	cacheKey := fmt.Sprintf("git:%s:%s", service, resource)

	if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Git backend: resource=%s, keyPath=%s\n", resource, keyPath)
	}

	// Check if we have cached the raw git credential response
	var rawCredentialResponse string
	var err error
	if cached, exists := cache.Get(cacheKey); exists {
		rawCredentialResponse = cached
		if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Git credential cache hit\n")
		}
	} else {
		if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Git credential cache miss, calling git credential helper\n")
		}
		// Cache miss - retrieve from git credential helper
		// For git, we need to extract username from resource if present
		cleanURL, username := parser.ParseGitURL(resource)
		if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Parsed URL: %s, username: %s\n", cleanURL, username)
		}
		rawCredentialResponse, err = getCredential(cleanURL, username)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve git credential for %s: %w", cleanURL, err)
		}

		if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Git credential retrieved successfully\n")
		}
		// Cache the raw git credential response directly
		cache.Set(cacheKey, rawCredentialResponse)
	}

	// Apply keyPath parsing to the raw credential response (same pattern as AWS)
	return parseGitCredential(rawCredentialResponse, keyPath)
}

// parseGitCredential parses git credential response and returns the requested part
// This is equivalent to extractJSONKey for AWS backend
func parseGitCredential(credentialResponse, keyPath string) (string, error) {
	if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Parsing git credential for keyPath: %s\n", keyPath)
	}

	// Parse the git credential format: "key=value\n" lines
	for _, line := range strings.Split(credentialResponse, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2) // Only split on first =, rest is value
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if key == keyPath {
			if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Found requested key '%s'\n", keyPath)
			}
			return value, nil
		}
	}

	if os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Key '%s' not found in git credential response\n", keyPath)
	}
	return "", fmt.Errorf("key '%s' not found in git credential response", keyPath)
}

// getCredential retrieves raw credentials from git credential fill.
func getCredential(url, user string) (string, error) {
	input := fmt.Sprintf("url=%s\n", url)
	if user != "" {
		input += fmt.Sprintf("username=%s\n", user)
	}
	input += "\n" // Important: git credential fill expects a blank line to terminate input

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git credential fill failed: %w", err)
	}

	return string(output), nil
}

// StoreCredential stores credentials using git credential helper
// url: the URL to store credentials for (can include user@ prefix, can be empty to prompt)
// username: username (optional if already in URL)
// Returns error if storage fails
func (b *GitBackend) StoreCredential(url, username string) error {
	// Prompt for URL if not provided
	if url == "" {
		fmt.Print("URL: ")
		if _, err := fmt.Scanln(&url); err != nil {
			return fmt.Errorf("error reading URL: %w", err)
		}
	}

	// Parse the URL to extract user if present and get clean URL
	cleanURL, userFromURL := parser.ParseGitURL(url)

	// Use username from parameter or extracted from URL
	if username == "" {
		username = userFromURL
	}

	// If we still don't have a username, prompt for it
	if username == "" {
		fmt.Print("Username: ")
		if _, err := fmt.Scanln(&username); err != nil {
			return fmt.Errorf("error reading username: %w", err)
		}
	}

	// Clear any existing credentials first
	if err := b.clearCredential(cleanURL, username); err != nil {
		// Ignore errors - credential might not exist
	}

	// Get credentials (this will prompt for password if needed)
	credentials, err := b.promptForCredentials(cleanURL, username)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Store the credentials
	if err := b.approveCredentials(credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

// clearCredential removes existing credentials
func (b *GitBackend) clearCredential(url, username string) error {
	input := fmt.Sprintf("url=%s\n", url)
	if username != "" {
		input += fmt.Sprintf("username=%s\n", username)
	}
	input += "\n"

	cmd := exec.Command("git", "credential", "reject")
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	return cmd.Run() // Ignore errors
}

// promptForCredentials prompts for credentials using git credential fill
func (b *GitBackend) promptForCredentials(url, username string) (string, error) {
	input := fmt.Sprintf("url=%s\n", url)
	if username != "" {
		input += fmt.Sprintf("username=%s\n", username)
	}
	input += "\n"

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// approveCredentials stores credentials using git credential approve
func (b *GitBackend) approveCredentials(credentials string) error {
	cmd := exec.Command("git", "credential", "approve")
	cmd.Stdin = strings.NewReader(credentials)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
