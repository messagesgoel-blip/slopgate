package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP058 flags SQL strings built with string concatenation or interpolation.
type SLP058 struct{}

func (SLP058) ID() string                { return "SLP058" }
func (SLP058) DefaultSeverity() Severity { return SeverityBlock }
func (SLP058) Description() string {
	return "SQL built with string concatenation"
}

var sqlConcatPattern = regexp.MustCompile(`(?is)\b(select|insert|update|delete|where|from|join)\b.*(\+|\$\{)|fmt\.Sprintf\s*\([^)]*(?:\b(select|insert|update|delete|where|from|join)\b[^)]*%[vTtbcdoqxXfFeEgGsp]|%[vTtbcdoqxXfFeEgGsp][^)]*\b(select|insert|update|delete|where|from|join)\b)[^)]*\)`)

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
			locs := sqlConcatPattern.FindAllStringSubmatchIndex(ln.Content, -1)
			for _, loc := range locs {
				if len(loc) > 0 {
					if loc[0] < 0 {
						continue
					}
					prefix := ln.Content[:loc[0]]
					if strings.Count(prefix, "`")%2 == 1 {
						continue
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
