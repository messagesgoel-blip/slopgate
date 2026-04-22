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

func isTestPath(path string) bool {
	p := strings.ReplaceAll(strings.ToLower(path), "\\", "/")
	return strings.Contains(p, "_test.") ||
		strings.Contains(p, ".test.") ||
		strings.Contains(p, ".spec.") ||
		strings.Contains("/"+p+"/", "/test/") ||
		strings.Contains("/"+p+"/", "/tests/") ||
		strings.Contains("/"+p+"/", "/testdata/")
}

func (r SLP052) Check(d *diff.Diff) []Finding {
	var out []Finding
	var prodDeleteCount int
	var testAdd bool
	for _, f := range d.Files {
		lowerPath := strings.ToLower(f.Path)
		isTest := isTestPath(lowerPath)
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
