package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP045 flags HTTP handlers that call DB functions without passing context.
//
// Rationale: Database operations should receive a context for proper timeout and cancellation
// handling. AI agents often forget to pass context from r.Context() to DB functions.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP045 struct{}

func (SLP045) ID() string                { return "SLP045" }
func (SLP045) DefaultSeverity() Severity { return SeverityWarn }
func (SLP045) Description() string {
	return "DB function called without context - use r.Context() for proper timeout handling"
}

// dbCallWithoutContextRe matches DB function calls that look like they need context.
var dbCallWithoutContextRe = regexp.MustCompile(`(?i)\.(Query|Exec|QueryContext|ExecContext|Ping|Prepare)\s*\(`)

// contextAssignmentRe matches if context is assigned from r.Context().
var contextAssignmentRe = regexp.MustCompile(`(?i)ctx\s*:?=\s*r\.Context\(\)`)

func (r SLP045) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		var addedContent strings.Builder
		var dbCallLines []diff.Line
		for _, line := range f.AddedLines() {
			addedContent.WriteString(line.Content)
			addedContent.WriteString("\n")
			if dbCallWithoutContextRe.MatchString(line.Content) {
				dbCallLines = append(dbCallLines, line)
			}
		}

		// If we found DB calls, check if context is present in the file.
		if len(dbCallLines) > 0 {
			content := addedContent.String()
			// Check for context usage
			hasContextAssignment := contextAssignmentRe.MatchString(content)
			hasContextParam := strings.Contains(content, "ctx context.Context")
			hasRContext := strings.Contains(content, "r.Context()")

			// Flag if no context is used at all
			if !hasContextAssignment && !hasContextParam && !hasRContext {
				for _, line := range dbCallLines {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  strings.TrimSpace(line.Content),
					})
				}
			}
		}
	}
	return out
}
