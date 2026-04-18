package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP027 flags async functions that throw synchronously instead of
// returning Promise.reject. Mixed error semantics force callers to
// handle both try/catch and .catch().
//
// Pattern: async function with throw before return Promise.
//
// Exempt: test files, docs.
type SLP027 struct{}

func (SLP027) ID() string                { return "SLP027" }
func (SLP027) DefaultSeverity() Severity { return SeverityWarn }
func (SLP027) Description() string {
	return "async function throws synchronously — use return Promise.reject for consistent error handling"
}

// slp027AsyncThrow matches async function with sync throw.
var slp027AsyncThrow = regexp.MustCompile(`(?i)async\s+function\s+\w+\s*\([^)]*\)\s*\{[^}]*throw\s+new\s+Error`)

// slp027PromiseThrow matches Promise-returning function with throw.
var slp027PromiseThrow = regexp.MustCompile(`(?i)function\s+\w+\s*\([^)]*\)\s*:\s*Promise[^{]*\{[^}]*throw\s+new\s+Error`)

func (r SLP027) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		if !strings.HasSuffix(lower, ".js") && !strings.HasSuffix(lower, ".ts") {
			continue
		}
		if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
			continue
		}

		for _, ln := range f.AddedLines() {
			clean := stripCommentAndStrings(ln.Content)
			if slp027AsyncThrow.MatchString(clean) || slp027PromiseThrow.MatchString(clean) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "async function throws synchronously — use return Promise.reject(err)",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}