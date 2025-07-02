package parser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SecretSource represents the parsed components of a secret string
type SecretSource struct {
	Backend  string
	Service  string // For cloud providers (sm, ps, kv, etc.)
	Resource string // The actual identifier (URL, name, ARN)
	KeyPath  string // Optional path for JSON extraction or specific credential part. Empty means raw content.
}

// ParseSecretString parses the input string into a SecretSource struct.
// It uses ":::" as the explicit delimiter for the optional KeyPath.
// Conventionally, the resource string should not contain ":::".
// Any string is now valid for KeyPath across all backends.
func ParseSecretString(s string) (SecretSource, error) {
	var keyPath string
	mainString := s

	// Step 1: Check for the explicit KeyPath delimiter ":::"
	keyPathParts := strings.SplitN(s, ":::", 2)
	if len(keyPathParts) == 2 {
		mainString = keyPathParts[0] // The part before ":::"
		keyPath = keyPathParts[1]    // The part after ":::" is the KeyPath
	}

	// Step 2: Split the mainString (without KeyPath) by the first colon to get backend and the rest
	parts := strings.SplitN(mainString, ":", 2)
	if len(parts) < 2 {
		return SecretSource{}, fmt.Errorf("invalid secret string format: %s. Expected at least 'backend:resource'", mainString)
	}

	backend := parts[0]
	remaining := parts[1] // This segment contains the service and resource

	secretSource := SecretSource{
		Backend: backend,
		KeyPath: keyPath, // Set the parsed KeyPath
	}

	switch backend {
	case "git":
		// Git format: git:repo_url[:::key_path]
		// The 'remaining' string here is the Git URL itself.

		// Normalize git URLs - handle both full URLs and short forms like user@host
		normalizedURL := normalizeGitURL(remaining)
		secretSource.Resource = normalizedURL

		// Validate the normalized URL
		u, err := url.Parse(secretSource.Resource)
		if err != nil {
			return SecretSource{}, fmt.Errorf("invalid Git URL in secret string: %w", err)
		}
		// Ensure the URL has a scheme after normalization
		if u.Scheme == "" {
			return SecretSource{}, fmt.Errorf("invalid Git URL scheme for resource '%s'", secretSource.Resource)
		}

	case "aws", "gcp", "azure":
		// These backends follow: backend:service:resource[:::key_path]
		// First, split off the service from the 'remaining' string.
		partsAfterBackend := strings.SplitN(remaining, ":", 2)
		if len(partsAfterBackend) < 2 {
			return SecretSource{}, fmt.Errorf("invalid %s secret string format: %s. Expected '%s:service:resource'", backend, mainString, backend)
		}
		secretSource.Service = partsAfterBackend[0]  // e.g., "sm", "ps", "kv"
		secretSource.Resource = partsAfterBackend[1] // The rest is the resource
		// The ":::" delimiter already handled the KeyPath separation, so no further heuristics needed here.

	default:
		return SecretSource{}, fmt.Errorf("unsupported backend: %s", backend)
	}

	return secretSource, nil
}

// normalizeGitURL handles different git URL formats and normalizes them
// Supports both full URLs (https://user@host/path) and short forms (user@host)
func normalizeGitURL(rawURL string) string {
	// If it already has a scheme, return as-is
	if strings.Contains(rawURL, "://") {
		return rawURL
	}

	// Add https:// to URLs without a scheme
	return "https://" + rawURL
}

// parseGitURL is a utility function that extracts username from Git URL if present and returns clean URL
// This is used by secretinit --store and other components that need to parse Git URLs
func parseGitURL(rawURL string) (string, string) {
	// Regex to match URLs with user@ prefix in both full and short forms
	// Matches: https://user@host, http://user@host, or user@host
	userURLRegex := regexp.MustCompile(`^(?:(https?://))?([^@]+)@(.+)$`)

	if matches := userURLRegex.FindStringSubmatch(rawURL); matches != nil {
		user := matches[2]     // username part
		hostPath := matches[3] // host and path part (without user)

		// Normalize the clean URL (without user) using existing function
		// normalizeGitURL will handle adding scheme if needed
		normalizedURL := normalizeGitURL(hostPath)

		return normalizedURL, user
	}

	// No user found, just normalize and return
	return normalizeGitURL(rawURL), ""
}

// ParseGitURL is a public wrapper for parseGitURL to extract username from Git URL if present and return clean URL
// This is used by other packages that need to parse Git URLs with user credentials
func ParseGitURL(rawURL string) (string, string) {
	return parseGitURL(rawURL)
}
