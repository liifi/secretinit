package backend

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitBackend implements the Backend interface for the Git credential manager.
type GitBackend struct{}

// RetrieveSecret retrieves a secret from the Git credential manager.
// The service parameter is ignored for git backend.
// The resource string is expected to be a URL, optionally with a username included
// (e.g., "https://user@github.com" or "https://github.com").
// The keyPath should be "username" or "password".
func (b *GitBackend) RetrieveSecret(service, resource, keyPath string) (string, error) {
	// Parse the resource string to extract URL and potential user
	url, user := parseURLForUser(resource)

	username, password, err := getCredential(url, user)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve git credential for %s: %w", url, err)
	}

	switch keyPath {
	case "username":
		return username, nil
	case "password":
		return password, nil
	}
	return "", fmt.Errorf("invalid key path for git backend: %s. Expected 'username' or 'password'", keyPath)
}

// parseURLForUser extracts username from URL if present and returns clean URL
// Only handles simple case of user@host, not complex password scenarios
func parseURLForUser(rawURL string) (string, string) {
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, "://") {
		// Handle full URLs like https://user@github.com/repo
		parts := strings.SplitN(rawURL, "://", 2)
		if len(parts) == 2 {
			scheme := parts[0]
			remainder := parts[1]

			if atIndex := strings.Index(remainder, "@"); atIndex != -1 {
				userPart := remainder[:atIndex]
				hostPart := remainder[atIndex+1:]

				// Return clean URL without credentials
				cleanURL := scheme + "://" + hostPart
				return cleanURL, userPart
			}
		}
	} else if strings.Contains(rawURL, "@") {
		// Handle simple case: user@host
		parts := strings.SplitN(rawURL, "@", 2)
		if len(parts) == 2 {
			return "https://" + parts[1], parts[0]
		}
	}
	return rawURL, ""
}

// ParseURLForUser is a public version for use by other packages
func ParseURLForUser(rawURL string) (string, string) {
	return parseURLForUser(rawURL)
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
