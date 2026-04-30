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

func isSourcePath(path string) bool {
	p := strings.ReplaceAll(strings.ToLower(path), "\\", "/")
	for _, ext := range []string{".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".java", ".rb", ".rs", ".php", ".c", ".cc", ".cpp", ".h", ".hpp", ".cs", ".swift", ".kt"} {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	for _, dir := range []string{"/src/", "/pkg/", "/internal/", "/cmd/", "/lib/", "/app/"} {
		if strings.Contains("/"+p, dir) {
			return true
		}
	}
	return false
}

func (r SLP052) Check(d *diff.Diff) []Finding {
	var out []Finding
	var prodDeleteCount int
	var prodAddCount int
	var testTouched bool
	for _, f := range d.Files {
		lowerPath := strings.ToLower(f.Path)
		isTest := isTestPath(lowerPath)
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete && !isTest && isSourcePath(lowerPath) {
					prodDeleteCount++
				}
				if ln.Kind == diff.LineAdd && !isTest && isSourcePath(lowerPath) {
					prodAddCount++
				}
				if isTest && (ln.Kind == diff.LineAdd || ln.Kind == diff.LineDelete) {
					testTouched = true
				}
			}
		}
	}
	if prodDeleteCount >= 3 && testTouched && prodAddCount < prodDeleteCount {
		out = append(out, Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			File:     "",
			Line:     0,
			Message:  r.Description(),
			Snippet:  "",
		})
	}
	return out
}
