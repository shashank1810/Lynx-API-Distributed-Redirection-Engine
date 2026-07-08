// Package validator provides reusable URL validation utilities.
package validator

import (
	"net/url"
	"strings"
)

// allowedSchemes lists the accepted URL schemes.
var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
}

// blockedHosts prevents redirection loops and internal abuse.
var blockedHosts = map[string]bool{
	"localhost":   true,
	"127.0.0.1":  true,
	"0.0.0.0":    true,
	"[::1]":      true,
}

// IsValidURL validates that a raw URL string is well-formed, uses an allowed
// scheme, and does not target a blocked host.
func IsValidURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	if !allowedSchemes[parsed.Scheme] {
		return false
	}

	host := parsed.Hostname()
	if host == "" {
		return false
	}

	if blockedHosts[host] {
		return false
	}

	return true
}

// NormalizeURL trims whitespace and ensures consistent formatting.
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	// Remove trailing slash for consistency, unless it's just the root path.
	if len(rawURL) > 1 && strings.HasSuffix(rawURL, "/") {
		u, err := url.Parse(rawURL)
		if err == nil && u.Path == "/" && u.RawQuery == "" && u.Fragment == "" {
			return rawURL
		}
		rawURL = strings.TrimRight(rawURL, "/")
	}
	return rawURL
}
