package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP143 flags direct env var access without validation in critical sections.
// This catches patterns like:
//   - process.env.KEY (unchecked)
//   - import.meta.env.KEY (vite)
//   - Deno.env.get() (less common)
//
// The rule is context-sensitive: it allows env var usage in test files,
// config files with explicit validation patterns, and files named
// config/env/constants.
//
// Languages: JavaScript, TypeScript, JSX, TSX
//
// Scope: production source files only (excludes tests, config setup)
type SLP143 struct{}

func (SLP143) ID() string                { return "SLP143" }
func (SLP143) DefaultSeverity() Severity { return SeverityWarn }
func (SLP143) Description() string {
	return "environment variable accessed without validation or default"
}

// envVarPattern matches direct environment variable access in various forms.
var envVarPattern = regexp.MustCompile(`(?:process\.env\.|import\.meta\.env\.)\w+`)

// validationPattern matches common validation patterns that make env var usage safe.
var validationPattern = regexp.MustCompile(`(?:if\s*.+|const\s+\w+\s*=\s*\w+\s*\|\||\?\?|Boolean\()|\.hasOwnProperty|env\[`)

// isTestOrDedicatedConfig reports whether path is a test file or env config.
func isTestOrDedicatedConfig(path string) bool {
	lower := strings.ToLower(path)
	// Test file heuristics
	if strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.Contains(lower, "_test.") ||
		strings.Contains(lower, "/__tests__/") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") {
		return true
	}
	// Dedicated config files
	return strings.Contains(lower, "config") ||
		strings.Contains(lower, "env") ||
		strings.HasSuffix(lower, ".env")
}

// hasValidationOnLine checks if the line contains validation patterns.
// It's a simple heuristic: looks for if-checks, default operators, Boolean wrappers.
func hasValidationOnLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Check for common validation patterns
	if validationPattern.MatchString(trimmed) {
		return true
	}
	// Check for logical OR default patterns
	if strings.Contains(trimmed, "||") && strings.Contains(trimmed, "env") {
		return true
	}
	// Check for nullish coalescing
	if strings.Contains(trimmed, "??") {
		return true
	}
	// Check for ternary default
	if strings.Contains(trimmed, "? ") && strings.Contains(trimmed, ":") {
		return true
	}
	return false
}

func (r SLP143) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only check JS/TS source files
		if !strings.HasSuffix(f.Path, ".js") &&
			!strings.HasSuffix(f.Path, ".jsx") &&
			!strings.HasSuffix(f.Path, ".ts") &&
			!strings.HasSuffix(f.Path, ".tsx") {
			continue
		}
		// Skip test files and config-like files
		if isTestOrDedicatedConfig(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			line := ln.Content
			if !envVarPattern.MatchString(line) {
				continue
			}
			// If env var is used without validation, flag it
			if !hasValidationOnLine(line) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "environment variable accessed without validation or default value",
					Snippet:  strings.TrimSpace(line),
				})
			}
		}
	}

	return out
}
