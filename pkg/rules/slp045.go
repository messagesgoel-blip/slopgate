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

// dbCallWithoutContextRe matches DB function calls that look like they need context
// but are NOT already using the Context variants (ExecContext, QueryContext).
var dbCallWithoutContextRe = regexp.MustCompile(`\.(Query|Exec|Ping|Prepare)\s*\(`)

// directContextRe matches direct context arguments like context.Background(),
// context.TODO(), or context.With* calls that are passed inline.
var directContextRe = regexp.MustCompile(`context\.(Background|TODO|WithDeadline|WithTimeout|WithCancel|WithValue)\s*\(`)

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

		lines := f.AddedLines()
		for i, line := range lines {
			if !dbCallWithoutContextRe.MatchString(line.Content) {
				continue
			}
			// Check if this specific call already passes a context argument.
			// e.g. db.Exec(ctx, ...) or db.Query(ctx, ...) already has context.
			if strings.Contains(line.Content, "ctx") || directContextRe.MatchString(line.Content) {
				continue
			}

			// Check a window around this line for context availability.
			start := i - 10
			if start < 0 {
				start = 0
			}
			end := i + 5
			if end > len(lines) {
				end = len(lines)
			}
			hasContext := false
			for j := start; j < end; j++ {
				c := lines[j].Content
				if contextAssignmentRe.MatchString(c) ||
					strings.Contains(c, "ctx context.Context") ||
					strings.Contains(c, "r.Context()") ||
					directContextRe.MatchString(c) {
					hasContext = true
					break
				}
			}
			if !hasContext {
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
	return out
}
