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
var queryWithoutLimitRe = regexp.MustCompile(`(?i)SELECT\b.*\bFROM\b`)

func (r SLP041) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		lines := f.AddedLines()
		inRawString := false
		var rawStartLine diff.Line
		var rawContent strings.Builder

		for _, line := range lines {
			content := line.Content

			// Track raw string literal state (backtick-delimited)
			backtickCount := strings.Count(content, "`")
			if backtickCount%2 == 1 {
				if !inRawString {
					// Starting a raw string
					inRawString = true
					rawStartLine = line
					rawContent.Reset()
					rawContent.WriteString(content)
				} else {
					// Closing a raw string
					rawContent.WriteString("\n")
					rawContent.WriteString(content)
					inRawString = false
					upper := strings.ToUpper(rawContent.String())
					if queryWithoutLimitRe.MatchString(upper) && !strings.Contains(upper, "LIMIT") {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     rawStartLine.NewLineNo,
							Message:  r.Description(),
							Snippet:  strings.TrimSpace(rawContent.String()),
						})
					}
				}
				continue
			}

			if inRawString {
				rawContent.WriteString("\n")
				rawContent.WriteString(content)
				continue
			}

			// Single-line check for quoted strings
			trimmed := strings.TrimSpace(content)
			upper := strings.ToUpper(trimmed)
			if strings.Contains(trimmed, "`") || strings.Contains(trimmed, "\"") {
				if queryWithoutLimitRe.MatchString(upper) && !strings.Contains(upper, "LIMIT") {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  trimmed,
					})
				}
			}
		}
	}
	return out
}
