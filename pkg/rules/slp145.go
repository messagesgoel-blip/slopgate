package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP145 flags hardcoded timeout values that lack contextual justification
// via comments. Timeouts that are too short (under 1s) or too long (over 30s)
// should have an explanation of why those values are chosen.
//
// Detected patterns:
//   - setTimeout, setInterval with numeric literals (ms)
//   - fetch/axios/timeout options with ms values
//   - database/connection timeouts (ms)
//   - HTTP client timeouts (ms)
//   - Go: context.WithTimeout, time.After, time.NewTimer (seconds)
//   - Python: time.sleep (seconds)
//   - Java: Thread.sleep (ms)
//
// Languages: JavaScript, TypeScript, Go, Python, Java
//
// Scope: all source files
type SLP145 struct{}

func (SLP145) ID() string                { return "SLP145" }
func (SLP145) DefaultSeverity() Severity { return SeverityWarn }
func (SLP145) Description() string {
	return "hardcoded timeout value lacks contextual justification"
}

// timeoutEntry pairs a regex with a unit multiplier (1 = ms, 1000 = seconds).
type timeoutEntry struct {
	pattern *regexp.Regexp
	unitMs  int // multiplier to convert captured value to ms
}

// timeoutPatterns matches common timeout usages with numeric literals.
var timeoutPatterns = []timeoutEntry{
	// setTimeout / setInterval (ms)
	{regexp.MustCompile(`\bsetTimeout\s*\(\s*[^,]+,\s*(\d+)\s*\)`), 1},
	{regexp.MustCompile(`\bsetInterval\s*\(\s*[^,]+,\s*(\d+)\s*\)`), 1},
	// fetch/axios timeout in ms
	{regexp.MustCompile(`timeout\s*:\s*(\d+)`), 1},
	{regexp.MustCompile(`timeout\s*=\s*(\d+)`), 1},
	// request/agent options (ms)
	{regexp.MustCompile(`(?:socket|connection|request)Timeout\s*[=:]\s*(\d+)`), 1},
	// Go: context.WithTimeout with numeric literal * time.Second (captures the multiplier)
	{regexp.MustCompile(`context\.WithTimeout\s*\(\s*[^,]+,\s*(\d+)\s*\*\s*time\.Second`), 1000},
	// Go: time.After / time.NewTimer with numeric literal * time.Second
	{regexp.MustCompile(`time\.(?:After|NewTimer)\s*\(\s*(\d+)\s*\*\s*time\.Second`), 1000},
	// Python: time.sleep (seconds)
	{regexp.MustCompile(`time\.sleep\s*\(\s*(\d+)\s*\)`), 1000},
	// Java: Thread.sleep (ms)
	{regexp.MustCompile(`Thread\.sleep\s*\(\s*(\d+)\s*\)`), 1},
}

// extremeTimeouts defines thresholds (in ms) for flagging.
const (
	veryShortTimeout = 1000  // < 1 second
	veryLongTimeout  = 30000 // > 30 seconds
)

// hasJustifyingComment checks if the line or the next line contains a
// comment that explains why this timeout value is chosen.
func hasJustifyingComment(lines []diff.Line, idx int) bool {
	// Check current line for trailing comment (but not URLs like http://)
	// Use LastIndex to find the rightmost "//" so comments after URLs are detected.
	current := lines[idx].Content
	if commentIdx := strings.LastIndex(current, "//"); commentIdx >= 0 {
		// Skip if "//" is part of a URL scheme (e.g., "http://", "https://")
		if !(commentIdx > 0 && current[commentIdx-1] == ':') {
			comment := strings.TrimSpace(current[commentIdx+2:])
			if len(comment) > 5 { // meaningful comment
				return true
			}
		}
	}
	// Also check for Python/shell-style "#" comments
	// Only treat # as a comment delimiter when it starts a line or follows
	// whitespace, not when it's part of a URL fragment or string content.
	if hashIdx := strings.Index(current, "#"); hashIdx >= 0 {
		if hashIdx == 0 || current[hashIdx-1] == ' ' || current[hashIdx-1] == '\t' {
			comment := strings.TrimSpace(current[hashIdx+1:])
			if len(comment) > 5 {
				return true
			}
		}
	}
	// Check previous line (post-patch: skip deleted lines)
	if idx > 0 {
		prev := lines[idx-1]
		if prev.Kind != diff.LineDelete {
			trimmed := strings.TrimSpace(prev.Content)
			if (strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#")) && len(trimmed) > 5 {
				return true
			}
		}
	}
	// Check next line (post-patch: accept both added and context lines)
	if idx+1 < len(lines) && lines[idx+1].Kind != diff.LineDelete {
		next := lines[idx+1].Content
		trimmed := strings.TrimSpace(next)
		if (strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#")) && len(trimmed) > 5 {
			return true
		}
	}
	return false
}

// isTimeoutExtreme returns true if value (in ms) is unusually short or long.
func isTimeoutExtreme(valueMs int) bool {
	return valueMs < veryShortTimeout || valueMs > veryLongTimeout
}

func (r SLP145) Check(d *diff.Diff) []Finding {
	if d == nil {
		return nil
	}
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only check code files (exclude markup, config)
		if strings.HasSuffix(f.Path, ".md") ||
			strings.HasSuffix(f.Path, ".json") ||
			strings.HasSuffix(f.Path, ".yaml") ||
			strings.HasSuffix(f.Path, ".yml") ||
			strings.HasSuffix(f.Path, ".toml") {
			continue
		}

		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				line := ln.Content
				// Check each timeout pattern
				for _, entry := range timeoutPatterns {
					if matches := entry.pattern.FindStringSubmatch(line); len(matches) > 1 {
						value, err := strconv.Atoi(matches[1])
						if err != nil {
							continue
						}
						// Convert to ms using the pattern's unit multiplier
						valueMs := value * entry.unitMs
						if isTimeoutExtreme(valueMs) && !hasJustifyingComment(h.Lines, i) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "extreme timeout value '" + matches[1] + "' lacks justification",
								Snippet:  strings.TrimSpace(line),
							})
						}
					}
				}
			}
		}
	}

	return out
}
