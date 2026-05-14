package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP058 flags SQL strings built with string concatenation or interpolation.
type SLP058 struct{}

// ID returns the rule identifier: "SLP058".
func (SLP058) ID() string { return "SLP058" }

// DefaultSeverity returns this rule's default severity.
func (SLP058) DefaultSeverity() Severity { return SeverityBlock }

// Description returns a short description of the SLP058 rule.
func (SLP058) Description() string {
	return "SQL built with string concatenation"
}

var sqlConcatPattern = regexp.MustCompile(`(?is)\b(select|insert|update|delete|where|from|join)\b.*(\+|\$\{)|fmt\.Sprintf\s*\([^)]*(?:\b(select|insert|update|delete|where|from|join)\b[^)]*%[vTtbcdoqxXfFeEgGsp]|%[vTtbcdoqxXfFeEgGsp][^)]*\b(select|insert|update|delete|where|from|join)\b)[^)]*\)`)

// Check implements the diff-aware SLP058 rule for SQL string concatenation.
func (r SLP058) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only check file types where SQL string concatenation is dangerous.
		ext := strings.ToLower(filepath.Ext(f.Path))
		if ext != ".go" && ext != ".js" && ext != ".jsx" && ext != ".ts" && ext != ".tsx" && !strings.HasSuffix(f.Path, ".py") {
			continue
		}
		for _, ln := range f.AddedLines() {
			// Skip matches inside Go backtick raw string literals
			// (e.g. regexp.MustCompile backtick strings that contain SQL keywords).
			// Only apply to Go files — JS/TS template literals also use backticks
			// but contain ${} interpolation that should still be flagged.
			isGo := strings.HasSuffix(strings.ToLower(f.Path), ".go")
			locs := sqlConcatPattern.FindAllStringSubmatchIndex(ln.Content, -1)
			for _, loc := range locs {
				if len(loc) > 0 {
					if loc[0] < 0 {
						continue
					}
					if isGo {
						prefix := ln.Content[:loc[0]]
						if strings.Count(prefix, "`")%2 == 1 {
							continue
						}
						if strings.Contains(prefix, "regexp.MustCompile(") ||
							strings.Contains(prefix, "regexp.Compile(") {
							continue
						}
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "SQL built with string concatenation — use parameterized queries",
						Snippet:  strings.TrimSpace(ln.Content),
					})
					break
				}
			}
		}
	}
	return out
}
