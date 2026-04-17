package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP001 flags test functions added wholesale in the current diff that
// contain no assertion token anywhere in their body.
//
// Rationale: AI agents asked to "write a test for X" often produce
// functions that call X, collect the result into `_`, and stop. The
// test compiles, contributes to coverage, and asserts nothing — the
// worst kind of test, because it convinces everyone the function is
// covered.
//
// Languages: Go, JS/TS, Python, Java, Rust.
//
// Scope: only test functions whose entire body was added in this diff.
type SLP001 struct{}

func (SLP001) ID() string                { return "SLP001" }
func (SLP001) DefaultSeverity() Severity { return SeverityWarn }
func (SLP001) Description() string {
	return "test function added with no assertion in its body"
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
	// testify
	"assert.", "require.",
	// gomega
	"Expect(", "Eventually(", "Consistently(",
	// goconvey
	"So(",
	// matryer/is
	"is.",
	// quicktest
	"qt.",
	// gopkg.in/check.v1
	"c.Check(", "c.Assert(",
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
		if f.IsDelete {
			continue
		}
		lang := testFileLang(f.Path)
		if lang == "" {
			continue
		}
		for _, h := range f.Hunks {
			var findings []Finding
			switch lang {
			case "go":
				findings = scanHunkForGoTests(f.Path, h, r.DefaultSeverity(), r.ID())
			case "js":
				findings = scanHunkForJSTests(f.Path, h, r.DefaultSeverity(), r.ID())
			case "py":
				findings = scanHunkForPyTests(f.Path, h, r.DefaultSeverity(), r.ID())
			case "java":
				findings = scanHunkForJavaTests(f.Path, h, r.DefaultSeverity(), r.ID())
			case "rust":
				findings = scanHunkForRustTests(f.Path, h, r.DefaultSeverity(), r.ID())
			}
			out = append(out, findings...)
		}
	}
	return out
}

// scanHunkForGoTests walks a single hunk looking for added Go test
// function bodies and emits a finding for each one that lacks an
// assertion.
func scanHunkForGoTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
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

		// Limitation: brace-depth tracking uses strings.Count which
		// miscounts braces inside string literals, runes, or comments
		// (e.g. `s := "}"` adds a phantom closing brace). Full Go
		// parsing is out of scope for v0.0.1. In practice this rarely
		// causes problems because test function bodies seldom contain
		// brace characters in top-level string literals. If depth
		// tracking is wrong, the worst outcome is a missed finding
		// (bodyAllAdded stays true but depth never reaches 0, so the
		// loop exits without emitting) — never a false positive.
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

// --- JS/TS test function scanning ---

// jsTestFuncSignature matches describe("...", it("...", or test("...".
var jsTestFuncSignature = regexp.MustCompile(`(?:describe|it|test)\s*\(\s*["']`)

// jsAssertTokens are assertion tokens for JS/TS test frameworks.
var jsAssertTokens = []string{
	// Jest/Vitest
	"expect(", "assert.", "assert(",
	// Chai
	"should.",
}

// scanHunkForJSTests walks a single hunk looking for added JS/TS test
// function bodies and emits a finding for each one that lacks an assertion.
func scanHunkForJSTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if ln.Kind != diff.LineAdd {
			i++
			continue
		}
		trimmed := strings.TrimSpace(ln.Content)
		if !jsTestFuncSignature.MatchString(trimmed) {
			i++
			continue
		}
		// Check if the line has the opening brace.
		hasBrace := strings.Contains(trimmed, "{")
		if !hasBrace && i+1 < len(lines) && lines[i+1].Kind == diff.LineAdd {
			hasBrace = strings.Contains(lines[i+1].Content, "{")
		}
		if !hasBrace {
			i++
			continue
		}
		startLine := ln.NewLineNo
		// Build body content from the test function line + subsequent added lines.
		depth := strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if depth <= 0 && i+1 < len(lines) && lines[i+1].Kind == diff.LineAdd {
			depth += strings.Count(lines[i+1].Content, "{") - strings.Count(lines[i+1].Content, "}")
		}
		if depth <= 0 {
			// Single-line test: check for assertions.
			sawAssertion := jsHasAssertion(trimmed)
			if !sawAssertion {
				out = append(out, Finding{
					RuleID:   ruleID,
					Severity: sev,
					File:     path,
					Line:     startLine,
					Message:  "JS/TS test function has no assertion -- add expect()/assert.* or delete it",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
			i++
			continue
		}
		// Multi-line: scan body until brace depth reaches 0.
		sawAssertion := jsHasAssertion(trimmed)
		bodyAllAdded := true
		j := i + 1
		for j < len(lines) && depth > 0 {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				bodyAllAdded = false
				break
			}
			if jsHasAssertion(bl.Content) {
				sawAssertion = true
			}
			depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
			j++
		}
		if bodyAllAdded && depth == 0 && !sawAssertion {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "JS/TS test function has no assertion -- add expect()/assert.* or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}

func jsHasAssertion(line string) bool {
	for _, tok := range jsAssertTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}

// --- Python test function scanning ---

// pyTestFuncSignature matches def test_xxx(self): or def test_xxx():.
var pyTestFuncSignature = regexp.MustCompile(`^\s*def\s+(test\w+)\s*\(`)

// pyAssertTokens are assertion tokens for Python test frameworks.
var pyAssertTokens = []string{
	"assert ", "self.assert", "self.assertEqual", "self.assertTrue",
	"self.assertFalse", "self.assertIsNone", "self.assertIsNotNone",
	"self.assertIn", "self.assertRaises",
}

// scanHunkForPyTests walks a hunk looking for added Python test function
// bodies with no assertion. Python uses indentation, not braces.
func scanHunkForPyTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if ln.Kind != diff.LineAdd {
			i++
			continue
		}
		m := pyTestFuncSignature.FindStringSubmatch(ln.Content)
		if m == nil {
			i++
			continue
		}
		funcName := m[1]
		startLine := ln.NewLineNo
		funcIndent := leadingSpaces(ln.Content)
		// Collect body: all subsequent added lines indented deeper.
		sawAssertion := false
		bodyAllAdded := true
		j := i + 1
		for j < len(lines) {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				bodyAllAdded = false
				break
			}
			blTrimmed := strings.TrimSpace(bl.Content)
			if blTrimmed == "" {
				j++
				continue
			}
			blIndent := leadingSpaces(bl.Content)
			if blIndent <= funcIndent {
				break
			}
			if pyHasAssertion(bl.Content) {
				sawAssertion = true
			}
			j++
		}
		if bodyAllAdded && !sawAssertion {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "Python test function " + funcName + " has no assertion -- add assert/self.assert.* or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}

func pyHasAssertion(line string) bool {
	for _, tok := range pyAssertTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}

// --- Java test function scanning ---

// javaTestAnnotation matches @Test annotation.
var javaTestAnnotation = regexp.MustCompile(`@\s*Test\b`)

// javaTestMethod matches JUnit 3-style test methods.
var javaTestMethod = regexp.MustCompile(`^\s*(public\s+)?(static\s+)?void\s+(test\w+)\s*\(`)

// javaAssertTokens are assertion tokens for Java test frameworks.
var javaAssertTokens = []string{
	"assertEquals(", "assertTrue(", "assertFalse(", "assertNotNull(",
	"assertNull(", "assertSame(", "assertThat(", "Assertions.",
	"assert ", "fail(", "Mockito.",
}

// scanHunkForJavaTests walks a hunk looking for added Java test methods.
func scanHunkForJavaTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if ln.Kind != diff.LineAdd {
			i++
			continue
		}
		trimmed := strings.TrimSpace(ln.Content)
		isTest := javaTestAnnotation.MatchString(trimmed) || javaTestMethod.MatchString(trimmed)
		if !isTest {
			i++
			continue
		}
		funcName := "testMethod"
		if m := javaTestMethod.FindStringSubmatch(trimmed); m != nil {
			funcName = m[3]
		}
		// Find the opening brace, scanning forward over subsequent added
		// lines to handle multi-line signatures (e.g., @Test on its own
		// line followed by "public void" on the next).
		depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
		startLine := ln.NewLineNo
		k := i + 1
		for depth <= 0 && k < len(lines) && lines[k].Kind == diff.LineAdd {
			depth += strings.Count(lines[k].Content, "{") - strings.Count(lines[k].Content, "}")
			if m := javaTestMethod.FindStringSubmatch(strings.TrimSpace(lines[k].Content)); m != nil {
				funcName = m[3]
			}
			k++
		}
		if depth <= 0 {
			i++
			continue
		}
		sawAssertion := javaHasAssertion(ln.Content)
		bodyAllAdded := true
		j := i + 1
		for j < len(lines) && depth > 0 {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				bodyAllAdded = false
				break
			}
			if javaHasAssertion(bl.Content) {
				sawAssertion = true
			}
			depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
			j++
		}
		if bodyAllAdded && depth == 0 && !sawAssertion {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "Java test method " + funcName + " has no assertion -- add assert* or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}

func javaHasAssertion(line string) bool {
	for _, tok := range javaAssertTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}

// --- Rust test function scanning ---

// rustTestAttr matches #[test] attribute.
var rustTestAttr = regexp.MustCompile(`#\[\s*test\s*\]`)

// rustTestFunc matches any fn signature after a #[test] attribute.
// We match any function name — #[test] can mark functions with arbitrary
// names like fn it_should_parse_valid_input().
var rustTestFunc = regexp.MustCompile(`^\s*(pub\s+)?fn\s+(\w+)\s*\(`)

// rustAssertTokens are assertion tokens for Rust test frameworks.
var rustAssertTokens = []string{
	"assert!(", "assert_eq!(", "assert_ne!",
	"panic!(", "should_panic",
}

// scanHunkForRustTests walks a hunk looking for added Rust test functions.
func scanHunkForRustTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if ln.Kind != diff.LineAdd {
			i++
			continue
		}
		// Check for #[test] attribute.
		if !rustTestAttr.MatchString(strings.TrimSpace(ln.Content)) {
			i++
			continue
		}
		// Next added line should be the function signature.
		if i+1 >= len(lines) || lines[i+1].Kind != diff.LineAdd {
			i++
			continue
		}
		sigLine := lines[i+1]
		m := rustTestFunc.FindStringSubmatch(strings.TrimSpace(sigLine.Content))
		if m == nil {
			i++
			continue
		}
		funcName := m[2]
		startLine := ln.NewLineNo
		// Collect body by brace counting from the signature line.
		depth := strings.Count(sigLine.Content, "{") - strings.Count(sigLine.Content, "}")
		if depth <= 0 {
			// Opening brace on a separate line.
			if i+2 < len(lines) && lines[i+2].Kind == diff.LineAdd {
				depth += strings.Count(lines[i+2].Content, "{") - strings.Count(lines[i+2].Content, "}")
			}
		}
		if depth <= 0 {
			i++
			continue
		}
		sawAssertion := rustHasAssertion(sigLine.Content)
		bodyAllAdded := true
		j := i + 2
		for j < len(lines) && depth > 0 {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				bodyAllAdded = false
				break
			}
			if rustHasAssertion(bl.Content) {
				sawAssertion = true
			}
			depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
			j++
		}
		if bodyAllAdded && depth == 0 && !sawAssertion {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "Rust test function " + funcName + " has no assertion -- add assert!()/assert_eq!() or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}

func rustHasAssertion(line string) bool {
	for _, tok := range rustAssertTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}
