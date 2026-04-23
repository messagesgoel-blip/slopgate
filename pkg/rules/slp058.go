package rules

import (
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

var sqlConcatPattern = regexp.MustCompile(`(?i)\b(select|insert|update|delete|where|from|join)\b.*(\+|\$\{)|fmt\.Sprintf.*\b(select|insert|update|delete|where|from|join)\b`)

func (r SLP058) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		for _, ln := range f.AddedLines() {
			if sqlConcatPattern.MatchString(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "SQL built with string concatenation — use parameterized queries",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
