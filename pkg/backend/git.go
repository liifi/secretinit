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
// The service parameter contains the username extracted by the parser (if any).
// The resource string is expected to be a normalized URL from the parser
// (e.g., "https://api.example.com" or "https://example.com").
// The keyPath should be "username" or "password".
func (b *GitBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	// The parser has already extracted the user and normalized the URL
	// service = username (if any), resource = clean normalized URL
	username, password, err := getCredential(resource, service)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve git credential for %s: %w", resource, err)
	}

	switch keyPath {
	case "username":
		return username, nil
	case "password":
		return password, nil
	}
	return "", fmt.Errorf("invalid key path for git backend: %s. Expected 'username' or 'password'", keyPath)
}

// getCredential retrieves credentials for a given URL and optional username using git credential fill.
func getCredential(url, user string) (string, string, error) {
	input := fmt.Sprintf("url=%s\n", url)
	if user != "" {
		input += fmt.Sprintf("username=%s\n", user)
	}
	input += "\n" // Important: git credential fill expects a blank line to terminate input

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("git credential fill failed: %w", err)
	}

	var username, password string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "username=") {
			username = strings.TrimPrefix(line, "username=")
		}
		if strings.HasPrefix(line, "password=") {
			password = strings.TrimPrefix(line, "password=")
		}
	}

	return username, password, nil
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
