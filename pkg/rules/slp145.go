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
//   - setTimeout, setInterval with numeric literals
//   - fetch/axios/timeout options with ms values
//   - database/connection timeouts
//   - HTTP client timeouts
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

// timeoutPatterns matches common timeout usages with numeric literals.
var timeoutPatterns = []*regexp.Regexp{
	// setTimeout / setInterval
	regexp.MustCompile(`setTimeout\s*\(\s*[^,]+,\s*(\d+)\s*\)`),
	regexp.MustCompile(`setInterval\s*\(\s*[^,]+,\s*(\d+)\s*\)`),
	// fetch/axios timeout in ms
	regexp.MustCompile(`timeout\s*:\s*(\d+)`),
	regexp.MustCompile(`timeout\s*=\s*(\d+)`),
	// request/agent options
	regexp.MustCompile(`(?:socket|connection|request)Timeout\s*[=:]\s*(\d+)`),
	// context.WithTimeout with numeric literal (Go)
	regexp.MustCompile(`context\.WithTimeout\s*\(\s*[^,]+,\s*(\d+)\s*\*`),
	// time.After / time.NewTimer with numeric (non-capturing group for func name)
	regexp.MustCompile(`time\.(?:After|NewTimer)\s*\(\s*(\d+)\s*`),
	// Python: time.sleep, requests timeout
	regexp.MustCompile(`time\.sleep\s*\(\s*(\d+)\s*\)`),
	// Java: Thread.sleep
	regexp.MustCompile(`Thread\.sleep\s*\(\s*(\d+)\s*\)`),
}

// extremeTimeouts defines thresholds (in ms) for flagging.
const (
	veryShortTimeout = 1000   // < 1 second
	veryLongTimeout  = 30000  // > 30 seconds
)

// hasJustifyingComment checks if the line or the next line contains a
// comment that explains why this timeout value is chosen.
func hasJustifyingComment(lines []diff.Line, idx int) bool {
	// Check current line for trailing comment
	current := lines[idx].Content
	if strings.Contains(current, "//") {
		parts := strings.SplitAfter(current, "//")
		if len(parts) > 1 {
			comment := strings.TrimSpace(parts[1])
			if len(comment) > 5 { // meaningful comment
				return true
			}
		}
	}
	// Check previous line
	if idx > 0 {
		prev := lines[idx-1].Content
		trimmed := strings.TrimSpace(prev)
		if strings.HasPrefix(trimmed, "//") && len(trimmed) > 5 {
			return true
		}
	}
	// Check next line
	if idx+1 < len(lines) && lines[idx+1].Kind == diff.LineAdd {
		next := lines[idx+1].Content
		trimmed := strings.TrimSpace(next)
		if strings.HasPrefix(trimmed, "//") && len(trimmed) > 5 {
			return true
		}
	}
	return false
}

// isTimeoutExtreme returns true if value is unusually short or long.
func isTimeoutExtreme(valueStr string) (bool, error) {
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return false, err
	}
	return value < veryShortTimeout || value > veryLongTimeout, nil
}

func (r SLP145) Check(d *diff.Diff) []Finding {
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
				for _, pattern := range timeoutPatterns {
					if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
						valueStr := matches[1]
						// Check if this is an extreme value needing justification
						if extreme, err := isTimeoutExtreme(valueStr); extreme && err == nil {
							// Look for justifying comment
							if !hasJustifyingComment(h.Lines, i) {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     ln.NewLineNo,
									Message:  "extreme timeout value '" + valueStr + "ms' lacks justification",
									Snippet:  strings.TrimSpace(line),
								})
							}
						}
					}
				}
			}
		}
	}

	return out
}
