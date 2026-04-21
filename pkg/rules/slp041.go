package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP041 flags SQL queries without LIMIT clause.
//
// Rationale: Queries without LIMIT can return unbounded result sets,
// leading to memory exhaustion and performance issues. AI agents often
// forget to add LIMIT to queries.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP041 struct{}

func (SLP041) ID() string                { return "SLP041" }
func (SLP041) DefaultSeverity() Severity { return SeverityWarn }
func (SLP041) Description() string {
	return "SQL query without LIMIT clause may return unbounded results"
}

// queryWithoutLimitRe matches SQL queries (SELECT) that don't have LIMIT.
// Looks for SELECT ... FROM ... ; patterns without LIMIT.
var queryWithoutLimitRe = regexp.MustCompile(`(?i)SELECT.*FROM.*;?\s*$`)

func (r SLP041) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		for _, line := range f.AddedLines() {
			content := strings.TrimSpace(line.Content)
			// Check if it's a SQL query string (contains SELECT and FROM)
			if strings.Contains(strings.ToUpper(content), "SELECT") &&
				strings.Contains(strings.ToUpper(content), "FROM") &&
				!strings.Contains(strings.ToUpper(content), "LIMIT") {
				// Only flag if it's a multiline string or raw string literal (likely SQL)
				if strings.Contains(content, "`") || strings.Contains(content, "\"") {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  content,
					})
				}
			}
		}
	}
	return out
}
