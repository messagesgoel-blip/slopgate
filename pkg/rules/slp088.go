package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP088 flags hardcoded secrets, credentials, and API keys in source code.
// This is a critical security issue that can lead to data breaches.
// Note: This rule scans source files (.js, .ts, .go, .py), not config files.
// Config files (.toml, .yml, .yaml, .json, .env) are intentionally skipped
// because they are designed for configuration management. The PR description
// listing SLP088 for "settings files" was misleading - the rule catches
// hardcoded credentials in actual source code (not configuration files).
type SLP088 struct{}

func (SLP088) ID() string                { return "SLP088" }
func (SLP088) DefaultSeverity() Severity { return SeverityBlock }
func (SLP088) Description() string {
	return "hardcoded credential detected - use environment variable or secret manager"
}

var (
	// Common credential patterns
	slp088CredentialPatterns = []*regexp.Regexp{
		// API keys
		regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|api-key)\s*[=:]\s*["'][A-Za-z0-9_\-]{20,}["']`),
		regexp.MustCompile(`(?i)["'][A-Za-z0-9_\-]{20,}["'].*(?:api[_-]?key|apikey|api-key)`),

		// Secret keys
		regexp.MustCompile(`(?i)(?:secret[_-]?key|secretkey|secret-key|app[_-]?secret)\s*[=:]\s*["'][A-Za-z0-9_\-]{16,}["']`),

		// Passwords
		regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[=:]\s*["'][^"']{4,}["']`),
		regexp.MustCompile(`(?i)["'][^"']{4,}["']\s*[:=]\s*(?:password|passwd|pwd)`),

		// Tokens
		regexp.MustCompile(`(?i)(?:token|auth[_-]?token)\s*[=:]\s*["'][A-Za-z0-9_\-\.]{16,}["']`),
		regexp.MustCompile(`(?i)(?:bearer|Bearer)\s+[A-Za-z0-9_\-\.]{20,}`),

		// AWS credentials
		regexp.MustCompile(`(?i)(?:aws[_-]?access[_-]?key|aws_access_key)\s*[=:]\s*["'][A-Z0-9]{20}["']`),
		regexp.MustCompile(`(?i)(?:aws[_-]?secret[_-]?key|aws_secret_key)\s*[=:]\s*["'][A-Za-z0-9/+=]{40}["']`),

		// Private keys
		regexp.MustCompile(`(?i)(?:-----BEGIN\s+(?:RSA|EC|DSA|OPENSSH)\s+PRIVATE\s+KEY-----|-----BEGIN\s+PRIVATE\s+KEY-----)`),

		// Generic high-entropy strings (long random-looking)
		regexp.MustCompile(`(?i)(?:key|secret|credential|password|token)\s*[=:]\s*["'][A-Za-z0-9/+=]{32,}["']`),
	}

	// Patterns that indicate we're in a config file (should be allowed)
	slp088ConfigPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:env|process\.env|os\.Getenv|os\.Getenv|config\.get|config\.getValue)`),
		regexp.MustCompile(`(?i)(?:dotenv|config\.env|\.env\.)`),
	}
)

// Check scans source files for hardcoded credentials and secrets.
func (r SLP088) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		fileLower := strings.ToLower(f.Path)

		// Skip config files (.toml, .yml, .yaml, .json, .env)
		// These files are for configuration, not hardcoded secrets
		if strings.Contains(fileLower, ".env") && !strings.Contains(fileLower, "example") {
			continue
		}
		if strings.Contains(fileLower, ".toml") || strings.Contains(fileLower, ".yml") ||
			strings.Contains(fileLower, ".yaml") || strings.Contains(fileLower, ".json") {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Skip if it's using environment variable pattern
				isEnvPattern := false
				for _, pattern := range slp088ConfigPatterns {
					if pattern.MatchString(content) {
						isEnvPattern = true
						break
					}
				}
				if isEnvPattern {
					continue
				}

				// Check against credential patterns
				for _, pattern := range slp088CredentialPatterns {
					if pattern.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "hardcoded credential detected - use environment variable (process.env.XXX) or secret manager instead",
							Snippet:  content,
						})
						break
					}
				}
			}
		}
	}
	return out
}
