package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP052 flags diffs that delete production code while also modifying
// tests. This heuristic catches the case where features are removed to
// make failing tests pass.
type SLP052 struct{}

func (SLP052) ID() string                { return "SLP052" }
func (SLP052) DefaultSeverity() Severity { return SeverityBlock }
func (SLP052) Description() string {
	return "production code deleted while tests modified — verify features not removed to fix tests"
}

func (r SLP052) Check(d *diff.Diff) []Finding {
	var out []Finding
	var prodDeleteCount int
	var testAdd bool
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		lowerPath := strings.ToLower(f.Path)
		isTest := strings.Contains(lowerPath, "_test.") ||
			strings.Contains(lowerPath, ".test.") ||
			strings.Contains(lowerPath, ".spec.") ||
			strings.Contains(lowerPath, "/test/") ||
			strings.Contains(lowerPath, "/tests/") ||
			strings.Contains(lowerPath, "/testdata/")
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete && !isTest {
					prodDeleteCount++
				}
				if ln.Kind == diff.LineAdd && isTest {
					testAdd = true
				}
			}
		}
	}
	if prodDeleteCount >= 3 && testAdd {
		out = append(out, Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			File:     "",
			Line:     0,
			Message:  "production code deleted while tests modified — verify features not removed to fix tests",
			Snippet:  "",
		})
	}
	return out
}
