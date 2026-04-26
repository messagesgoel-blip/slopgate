package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP026 flags SQL queries checking for NULL without excluding sentinel values.
// Pattern: WHERE hash IS NOT NULL without AND hash != 'marker' exclusion.
//
// Exempt: test files, docs.
type SLP026 struct{}

func (SLP026) ID() string                { return "SLP026" }
func (SLP026) DefaultSeverity() Severity { return SeverityWarn }
func (SLP026) Description() string {
	return "SQL NULL check without sentinel exclusion — consider excluding placeholder values"
}

func (r SLP026) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
			continue
		}
		// Check SQL files and JS/TS files with embedded SQL.
		isSQL := strings.HasSuffix(lower, ".sql")
		isJS := strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".ts")
		if !isSQL && !isJS {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := ln.Content
			// Don't strip comments for SQL - they're rare in queries
			// Check for IS NOT NULL pattern on hash/value without sentinel exclusion.
			hasNotNull := strings.Contains(strings.ToUpper(content), "IS NOT NULL")
			hasHashOrValue := strings.Contains(content, "hash") || strings.Contains(content, "value") || strings.Contains(content, "Hash") || strings.Contains(content, "Value")
			hasSentinelExclusion := strings.Contains(content, "!=") || strings.Contains(content, "<>") || strings.Contains(strings.ToLower(content), "not like") || strings.Contains(strings.ToLower(content), "not in")

			if hasNotNull && hasHashOrValue && !hasSentinelExclusion {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "SQL NULL check without sentinel exclusion — exclude 'folder-marker' or similar",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
