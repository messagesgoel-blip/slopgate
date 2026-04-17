package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP002 flags tautological assertions in test files — assertions that
// compare a value to itself, always passing regardless of actual
// behaviour. This is a common AI slop pattern: the model fills in both
// sides of an assertion with the same placeholder variable.
//
// Detected patterns:
//   - Go/testify: assert.Equal(t, x, x), require.Equal(t, x, x),
//     assert.True(t, true), assert.False(t, false)
//   - JS/TS:       expect(x).toBe(x), expect(x).toEqual(x),
//     assert.strictEqual(x, x)
//   - Python:      self.assertEqual(a, a), self.assertIs(a, a)
//   - Java/JUnit:  assertEquals(x, x), assertTrue(true), assertFalse(false),
//     assertThat(x).isEqualTo(x)
//   - Rust:        assert_eq!(x, x), assert!(true)
type SLP002 struct{}

func (SLP002) ID() string                { return "SLP002" }
func (SLP002) DefaultSeverity() Severity { return SeverityBlock }
func (SLP002) Description() string {
	return "tautological assertion compares a value to itself"
}

// ---------------------------------------------------------------------------
// Test-file heuristics
// ---------------------------------------------------------------------------

// isGoTestFile reports whether path is a Go test file.
func isGoTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

// isJSTestFile reports whether path is a JS/TS test file.
func isJSTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.")
}

// isPythonTestFile reports whether path is a Python test file.
func isPythonTestFile(path string) bool {
	base := path
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		base = path[idx+1:]
	}
	return strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") ||
		strings.HasSuffix(base, "_test.py")
}

// ---------------------------------------------------------------------------
// Go/testify patterns
// ---------------------------------------------------------------------------

// goEqualRe matches assert.Equal(t, X, Y) / require.Equal(t, X, Y) / etc.
// Group 1: assert/require, Group 2: method name, Group 3: t-var, Group 4: first arg, Group 5: second arg.
var goEqualRe = regexp.MustCompile(
	`\b(assert|require)\.(Equal|Exactly)\(\s*(\w+)\s*,\s*(\w+)\s*,\s*(\w+)\s*[,)]`,
)

// goTrueFalseRe matches assert.True(t, true) / assert.False(t, false).
// Group 1: assert/require, Group 2: True/False, Group 3: literal that makes it tautological.
var goTrueFalseRe = regexp.MustCompile(
	`\b(assert|require)\.(True|False)\(\s*\w+\s*,\s*(true|false)\s*\)`,
)

// ---------------------------------------------------------------------------
// JS/TS patterns
// ---------------------------------------------------------------------------

// jsExpectRe matches expect(X).toBe(X) / expect(X).toEqual(X) / etc.
// Group 1: first X, Group 2: matcher name, Group 3: second X.
var jsExpectRe = regexp.MustCompile(
	`\bexpect\(\s*(\w+)\s*\)\.(toBe|toEqual|toStrictEqual|deepEqual)\(\s*(\w+)\s*\)`,
)

// jsAssertStrictRe matches assert.strictEqual(X, X) / assert.deepStrictEqual(X, X).
// Group 1: method, Group 2: first arg, Group 3: second arg.
var jsAssertStrictRe = regexp.MustCompile(
	`\bassert\.(strictEqual|deepStrictEqual|deepEqual)\(\s*(\w+)\s*,\s*(\w+)\s*[,)]`,
)

// ---------------------------------------------------------------------------
// Python patterns
// ---------------------------------------------------------------------------

// pySelfAssertRe matches self.assertEqual(X, X) / self.assertIs(X, X) / etc.
// Group 1: method, Group 2: first arg, Group 3: second arg.
var pySelfAssertRe = regexp.MustCompile(
	`\bself\.assert(Equal|Is|AlmostEqual|CountEqual|ListEqual|TupleEqual|SetEqual|DictEqual)\(\s*(\w+)\s*,\s*(\w+)\s*[,)]`,
)

// ---------------------------------------------------------------------------
// Java / JUnit patterns
// ---------------------------------------------------------------------------

// javaAssertEqualsRe matches assertEquals(x, x), assertSame(x, x), etc.
// Group 1: method, Group 2: first arg, Group 3: second arg.
var javaAssertEqualsRe = regexp.MustCompile(
	`\b(assertEquals|assertSame|assertArrayEquals|assertLinesMatch)\(\s*(\w+)\s*,\s*(\w+)\s*[,)]`,
)

// javaAssertTrueFalseRe matches assertTrue(true) / assertFalse(false).
var javaAssertTrueFalseRe = regexp.MustCompile(
	`\b(assertTrue|assertFalse)\(\s*(true|false)\s*\)`,
)

// javaAssertThatRe matches assertThat(x).isEqualTo(x) / assertThat(x).isSameInstanceAs(x).
// Group 1: first x, Group 2: matcher, Group 3: second x.
var javaAssertThatRe = regexp.MustCompile(
	`\bassertThat\(\s*(\w+)\s*\)\.(isEqualTo|isSameInstanceAs|isSameAs)\(\s*(\w+)\s*\)`,
)

// ---------------------------------------------------------------------------
// Rust patterns
// ---------------------------------------------------------------------------

// rustAssertEqRe matches assert_eq!(x, x).
// Group 1: first arg, Group 2: second arg.
var rustAssertEqRe = regexp.MustCompile(
	`\bassert_eq!\(\s*(\w+)\s*,\s*(\w+)\s*[,)]`,
)

// rustAssertTrueRe matches assert!(true).
var rustAssertTrueRe = regexp.MustCompile(
	`\bassert!\(\s*(true|false)\s*\)`,
)

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

func (r SLP002) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		lang := testFileLang(f.Path)
		if lang == "" {
			continue
		}
		for _, ln := range f.AddedLines() {
			findings := checkLine(f.Path, ln, lang, r.DefaultSeverity())
			out = append(out, findings...)
		}
	}
	return out
}

// testFileLang returns the test-file language if the path is a test file,
// otherwise an empty string.
func testFileLang(path string) string {
	if isGoTestFile(path) {
		return "go"
	}
	if isJSTestFile(path) {
		return "js"
	}
	if isPythonTestFile(path) {
		return "py"
	}
	if isJavaTestFile(path) {
		return "java"
	}
	if isRustTestFile(path) {
		return "rust"
	}
	return ""
}

// checkLine inspects one added line and returns findings for any
// tautological assertion patterns matching the given language.
func checkLine(file string, ln diff.Line, lang string, sev Severity) []Finding {
	var out []Finding
	trimmed := strings.TrimSpace(ln.Content)

	switch lang {
	case "go":
		out = append(out, checkGoLine(file, ln, trimmed, sev)...)
	case "js":
		out = append(out, checkJSLine(file, ln, trimmed, sev)...)
	case "py":
		out = append(out, checkPYLine(file, ln, trimmed, sev)...)
	case "java":
		out = append(out, checkJavaLine(file, ln, trimmed, sev)...)
	case "rust":
		out = append(out, checkRustLine(file, ln, trimmed, sev)...)
	}
	return out
}

func checkGoLine(file string, ln diff.Line, trimmed string, sev Severity) []Finding {
	var out []Finding

	// assert.Equal(t, x, x) / require.Equal(t, x, x)
	if m := goEqualRe.FindStringSubmatch(trimmed); m != nil {
		if m[4] == m[5] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  m[1] + "." + m[2] + ": both sides are the same identifier " + m[4],
				Snippet:  trimmed,
			})
		}
	}

	// assert.True(t, true) / assert.False(t, false)
	if m := goTrueFalseRe.FindStringSubmatch(trimmed); m != nil {
		expected := "true"
		if m[2] == "False" {
			expected = "false"
		}
		if m[3] == expected {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  m[1] + "." + m[2] + ": tautological assertion with literal " + m[3],
				Snippet:  trimmed,
			})
		}
	}

	return out
}

func checkJSLine(file string, ln diff.Line, trimmed string, sev Severity) []Finding {
	var out []Finding

	// expect(x).toBe(x) / expect(x).toEqual(x) / etc.
	if m := jsExpectRe.FindStringSubmatch(trimmed); m != nil {
		if m[1] == m[3] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "expect()." + m[2] + "(): both sides are the same identifier " + m[1],
				Snippet:  trimmed,
			})
		}
	}

	// assert.strictEqual(x, x) / assert.deepStrictEqual(x, x)
	if m := jsAssertStrictRe.FindStringSubmatch(trimmed); m != nil {
		if m[2] == m[3] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "assert." + m[1] + "(): both sides are the same identifier " + m[2],
				Snippet:  trimmed,
			})
		}
	}

	return out
}

func checkPYLine(file string, ln diff.Line, trimmed string, sev Severity) []Finding {
	var out []Finding

	// self.assertEqual(a, a) / self.assertIs(a, a) / etc.
	if m := pySelfAssertRe.FindStringSubmatch(trimmed); m != nil {
		if m[2] == m[3] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "self.assert" + m[1] + "(): both sides are the same identifier " + m[2],
				Snippet:  trimmed,
			})
		}
	}

	return out
}

func checkJavaLine(file string, ln diff.Line, trimmed string, sev Severity) []Finding {
	var out []Finding

	// assertEquals(x, x) / assertSame(x, x) / etc.
	if m := javaAssertEqualsRe.FindStringSubmatch(trimmed); m != nil {
		if m[2] == m[3] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  m[1] + "(): both sides are the same identifier " + m[2],
				Snippet:  trimmed,
			})
		}
	}

	// assertTrue(true) / assertFalse(false)
	if m := javaAssertTrueFalseRe.FindStringSubmatch(trimmed); m != nil {
		expected := "true"
		if m[1] == "assertFalse" {
			expected = "false"
		}
		if m[2] == expected {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  m[1] + "(): tautological assertion with literal " + m[2],
				Snippet:  trimmed,
			})
		}
	}

	// assertThat(x).isEqualTo(x)
	if m := javaAssertThatRe.FindStringSubmatch(trimmed); m != nil {
		if m[1] == m[3] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "assertThat()." + m[2] + "(): both sides are the same identifier " + m[1],
				Snippet:  trimmed,
			})
		}
	}

	return out
}

func checkRustLine(file string, ln diff.Line, trimmed string, sev Severity) []Finding {
	var out []Finding

	// assert_eq!(x, x)
	if m := rustAssertEqRe.FindStringSubmatch(trimmed); m != nil {
		if m[1] == m[2] {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "assert_eq!(): both sides are the same identifier " + m[1],
				Snippet:  trimmed,
			})
		}
	}

	// assert!(true) / assert!(false)
	if m := rustAssertTrueRe.FindStringSubmatch(trimmed); m != nil {
		if m[1] == "true" {
			out = append(out, Finding{
				RuleID:   "SLP002",
				Severity: sev,
				File:     file,
				Line:     ln.NewLineNo,
				Message:  "assert!(): tautological assertion with literal true",
				Snippet:  trimmed,
			})
		}
	}

	return out
}
