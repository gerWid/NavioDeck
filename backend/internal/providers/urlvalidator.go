package providers

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// blockedSchemes lists URL schemes that are not HTTP/HTTPS and must not be
// fetched, as they could be used for SSRF to access local resources (e.g.
// file://, gopher://, dict://, ftp://).
var blockedSchemes = map[string]bool{
	"file":   true,
	"gopher": true,
	"dict":   true,
	"ftp":    true,
	"ftps":   true,
	"ldap":   true,
	"ldaps":  true,
	"sftp":   true,
	"tftp":   true,
}

// blockedHosts lists hostnames that must never be fetched even over HTTP/HTTPS.
// RFC-1918 / private LAN ranges are intentionally NOT blocked because this is a
// LAN dashboard that legitimately fetches from local servers (iCal, RSS, etc.).
// We only block loopback and the cloud metadata endpoint.
var blockedHosts = map[string]bool{
	"localhost":       true,
	"metadata.google": true,
}

// blockedIPs lists individual IP addresses (as strings) that are always blocked.
// 169.254.169.254 is the AWS/GCP/Azure instance metadata endpoint.
var blockedIPs = []string{
	"127.0.0.1",
	"::1",
	"169.254.169.254",
}

// ValidateURL checks that a URL is safe to fetch in the context of this
// local-LAN dashboard. It:
//   - only allows http:// and https:// schemes
//   - blocks file://, gopher://, dict://, ftp://, and similar dangerous schemes
//   - blocks "localhost" and 127.x.x.x to prevent container-internal access
//   - blocks 169.254.169.254 (cloud instance-metadata endpoint)
//   - allows all other hostnames / IP ranges (RFC-1918 is OK for a LAN tool)
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		if blockedSchemes[scheme] {
			return fmt.Errorf("URL scheme %q is not allowed", scheme)
		}
		return fmt.Errorf("URL scheme %q is not allowed: only http and https are supported", scheme)
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("URL has no host")
	}

	if blockedHosts[host] {
		return fmt.Errorf("host %q is not allowed", host)
	}

	// Block 127.x.x.x loopback range explicitly (in addition to "localhost").
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() {
			return fmt.Errorf("loopback address %q is not allowed", host)
		}
		// Block cloud instance-metadata IP.
		for _, blocked := range blockedIPs {
			if ip.Equal(net.ParseIP(blocked)) {
				return fmt.Errorf("IP address %q is not allowed", host)
			}
		}
	}

	return nil
}
