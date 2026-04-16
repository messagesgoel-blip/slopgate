package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP001 flags Go test functions added wholesale in the current diff
// that contain no assertion token anywhere in their body.
//
// Rationale: AI agents asked to "write a test for X" often produce
// functions that call X, collect the result into `_`, and stop. The
// test compiles, contributes to coverage, and asserts nothing — the
// worst kind of test, because it convinces everyone the function is
// covered.
//
// v0.0.1 scope: Go only, standalone test functions (not testify suite
// methods), and only test functions whose entire body was added in
// this diff. Other languages and incremental changes land in v0.0.2.
type SLP001 struct{}

func (SLP001) ID() string                { return "SLP001" }
func (SLP001) DefaultSeverity() Severity { return SeverityWarn }
func (SLP001) Description() string {
	return "Go test function added with no assertion in its body"
}

// safetyTestNamePattern matches common test-name suffixes that signal
// "this test is checking an invariant by not panicking" rather than
// asserting a value. These are legitimate no-assertion tests.
var safetyTestNamePattern = regexp.MustCompile(`(?i)(NilSafe|NoPanic|DoesNotPanic|_Panic$|_Safe$)`)

// isSafetyTestName reports whether a test function name looks like a
// panic-safety or nil-safety test.
func isSafetyTestName(name string) bool {
	return safetyTestNamePattern.MatchString(name)
}

// testFuncSignature matches an added line that opens a top-level Go
// test function. The opening brace must be on the same line — the
// canonical gofmt style — which makes brace-depth tracking below safe.
// Group 1 captures the function name; group 2 captures the testing.T
// parameter name (usually "t" but could be anything).
var testFuncSignature = regexp.MustCompile(`^func\s+(Test\w+)\s*\(\s*(\w+)\s*\*testing\.T\s*\)\s*\{`)

// libraryAssertTokens are assertion tokens that come from third-party
// test libraries and do not depend on the testing.T parameter name.
var libraryAssertTokens = []string{
	"assert.", "require.",
	"Expect(", "Eventually(", "Consistently(",
	"So(",
}

// hasAssertion reports whether the line contains at least one assertion.
// tVar is the testing.T parameter name captured from the function
// signature (usually "t"). Only calls on that specific variable count
// as assertions — `err.Error()` does not.
func hasAssertion(line, tVar string) bool {
	tSuffixes := []string{".Error(", ".Errorf(", ".Fatal(", ".Fatalf(", ".FailNow(", ".Fail("}
	for _, s := range tSuffixes {
		if strings.Contains(line, tVar+s) {
			return true
		}
	}
	for _, tok := range libraryAssertTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}

// isTopLevelSkipStatement reports whether the given added line is a
// bare test-skip statement at the statement level, scoped to the
// actual testing.T parameter. The leading-whitespace anchor means
// `if cond { t.Skip("x") }` does not match — only statement-level
// skips at the outer block of the function body.
func isTopLevelSkipStatement(line, tVar string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, tVar+".Skip(") ||
		strings.HasPrefix(trimmed, tVar+".SkipNow(") ||
		strings.HasPrefix(trimmed, tVar+".Skipf(")
}

func (r SLP001) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		for _, h := range f.Hunks {
			findings := scanHunkForTests(f.Path, h, r.DefaultSeverity(), r.ID())
			out = append(out, findings...)
		}
	}
	return out
}

// scanHunkForTests walks a single hunk looking for added Go test
// function bodies and emits a finding for each one that lacks an
// assertion.
func scanHunkForTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines

	i := 0
	for i < len(lines) {
		ln := lines[i]
		if ln.Kind != diff.LineAdd {
			i++
			continue
		}
		m := testFuncSignature.FindStringSubmatch(strings.TrimLeft(ln.Content, " \t"))
		if m == nil {
			i++
			continue
		}
		// Collect the body by brace-counting over subsequent added
		// lines. Any non-added line before the closing brace disqualifies
		// the detection — we can only reason about test bodies that are
		// wholly new.
		funcName, tVar := m[1], m[2]
		if isSafetyTestName(funcName) {
			// Intentional no-assertion test — advance past the signature
			// and continue.
			i++
			continue
		}

		depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
		startLine := ln.NewLineNo
		sawAssertion := hasAssertion(ln.Content, tVar)
		sawTopLevelSkip := false
		bodyAllAdded := true

		j := i + 1
		for j < len(lines) && depth > 0 {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				bodyAllAdded = false
				break
			}
			if hasAssertion(bl.Content, tVar) {
				sawAssertion = true
			}
			// A top-level t.Skip(...) at brace depth 1 (directly
			// inside the function body) marks the test as an
			// intentional scaffold — inert by design, not slop.
			if depth == 1 && isTopLevelSkipStatement(bl.Content, tVar) {
				sawTopLevelSkip = true
			}
			depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
			j++
		}

		if bodyAllAdded && depth == 0 && !sawAssertion && !sawTopLevelSkip {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "test function " + funcName + " has no assertion — add t.Error/t.Fatal/assert.* or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}

		// Advance past the function body (or to j if disqualified).
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}
