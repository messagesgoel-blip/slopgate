package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP005 flags test-runner exclusivity markers (it.only, describe.only,
// fdescribe, fit, test.only) committed on added lines. These markers
// are harmless in a local workflow — they let you focus one test while
// debugging — but catastrophic if merged: they silently skip the rest
// of your suite.
//
// AI-generated tests commonly emit .only during focused iteration and
// leave the marker in when the task was "get this test passing".
type SLP005 struct{}

func (SLP005) ID() string                { return "SLP005" }
func (SLP005) DefaultSeverity() Severity { return SeverityBlock }
func (SLP005) Description() string {
	return "test-runner focus marker (.only / fdescribe / fit) committed"
}

// slp005Patterns must match only test-runner idioms, not arbitrary
// `.only` access on unrelated objects. We pin the prefix to `it`,
// `describe`, `test`, `context`, or their focused aliases.
var slp005Patterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(it|describe|test|context)\.only\s*\(`),
	regexp.MustCompile(`\b(fdescribe|fit|ftest|fcontext)\s*\(`),
}

// isJSTestFilePath reports whether the path is a JS/TS test file
// where `.only` / `fdescribe` / `fit` are meaningful test-runner
// idioms (Jest, Mocha, Jasmine, Vitest).
//
// Python is deliberately excluded: `fit(` is a common legitimate
// function call in Python data-science code (sklearn model.fit(),
// scaler.fit(), pipeline.fit_transform()), and Python's focus
// mechanisms (@pytest.mark.only, -k flag) don't use the same
// syntax as JS test runners. Including Python would cause blocking
// false positives on any ML test that calls .fit().
//
// Go test files are also excluded: Go's testing package has no
// `.only` concept, and Go test sources often embed JS fixtures in
// raw string literals.
func isJSTestFilePath(path string) bool {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, ".test."):
		return true
	case strings.Contains(lower, ".spec."):
		return true
	case strings.Contains(lower, "/__tests__/"):
		return true
	}
	return false
}

func (r SLP005) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isJSTestFilePath(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			for _, p := range slp005Patterns {
				if p.MatchString(ln.Content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "test focus marker committed — other tests will be silently skipped",
						Snippet:  strings.TrimSpace(ln.Content),
					})
					break
				}
			}
		}
	}
	return out
}
