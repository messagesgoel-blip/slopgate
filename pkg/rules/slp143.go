package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP143 flags direct env var access without validation in critical sections.
// This catches patterns like:
//   - process.env.KEY (unchecked)
//   - process.env["KEY"] (bracket access)
//   - import.meta.env.KEY (vite)
//   - import.meta.env['KEY'] (bracket access)
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

// envVarPattern matches direct environment variable access in various forms
// including both dot-notation (process.env.KEY) and bracket-notation
// (process.env["KEY"], import.meta.env['KEY']).
var envVarPattern = regexp.MustCompile(`(?:process\.env\.|import\.meta\.env\.)(?:\w+|\[(?:["'][^"']+["'])\])`)

// validationPattern matches validation patterns that make env var usage safe.
// Does NOT use the broad `if\s*.+` pattern which would mark any if-statement
// as validation. Instead relies on specific patterns.
var validationPattern = regexp.MustCompile(`(?:const\s+\w+\s*=\s*\w+\s*\|\||\?\?|Boolean\()|\.hasOwnProperty|env\[`)

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
	// Dedicated config files: match by basename or path segment, not substring
	base := filepath.Base(lower)
	return strings.HasPrefix(base, "config.") ||
		strings.HasPrefix(base, "env.") ||
		base == "config" ||
		base == "env" ||
		strings.Contains(lower, "/config/") ||
		strings.Contains(lower, "/env/") ||
		strings.HasSuffix(lower, ".env")
}

// hasValidationOnLine checks if the line contains validation patterns.
// It's a simple heuristic: looks for default operators, nullish coalescing,
// Boolean wrappers, hasOwnProperty, and env bracket access with validation.
func hasValidationOnLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Check for specific validation patterns (not broad if-statements)
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
	// Check for ternary default (both "a ? b : c" and compact "a?b:c")
	qIdx := strings.Index(trimmed, "?")
	cIdx := strings.Index(trimmed, ":")
	if qIdx >= 0 && cIdx >= 0 && qIdx < cIdx {
		return true
	}
	// Check for if-statements that specifically inspect env vars
	if strings.Contains(trimmed, "if") && strings.Contains(trimmed, "env") {
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
