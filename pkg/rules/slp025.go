package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP025 flags URL-building by string concatenation without validation.
// Pattern: `${URL}${path}` or `${BASE_URL}${...}` without path validation.
//
// Exempt: test files, docs.
type SLP025 struct{}

func (SLP025) ID() string                { return "SLP025" }
func (SLP025) DefaultSeverity() Severity { return SeverityWarn }
func (SLP025) Description() string {
	return "URL built by concatenating path without validation — could produce malformed URLs"
}

func (r SLP025) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		if !strings.HasSuffix(lower, ".js") && !strings.HasSuffix(lower, ".ts") {
			continue
		}
		if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
			continue
		}

		for _, ln := range f.AddedLines() {
			// Check for URL concatenation pattern: ${...URL...}${...Path...} or ${BASE_URL}${...}
			content := ln.Content
			hasURL := strings.Contains(content, "URL") || strings.Contains(content, "BASE")
			hasPath := strings.Contains(content, "Path") || strings.Contains(content, "path")
			hasTemplate := strings.Contains(content, "${") && strings.Contains(content, "}${")
			if hasURL && hasPath && hasTemplate {
				// This looks like ${URL}${Path} concatenation without validation.
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "URL concatenation without path validation — ensure path starts with '/'",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
