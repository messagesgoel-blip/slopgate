package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP011 flags test functions whose body is entirely assertion calls
// with no meaningful arrange/logic. This catches AI-generated tests that
// look like "assert.Equal(t, 1, 1)" with no actual test logic.
//
// Languages: Go only. Non-Go languages don't have the same function-span
// detection needed for assert-only test body analysis.
//
// The key distinction from SLP010 (incremental no-assertion):
//   - SLP010: AI edited an existing test, added setup but no assertion
//   - SLP011: Entire test body is only assertions (assert-only test body)
//
// The one exception is a single variable assignment for the "arrange" value,
// e.g. "got := Foo()" followed by an assertion - that pattern is OK.
//
// SLP011 complements SLP001 (new test with no assertion):
//   - SLP001: New test that calls something but asserts nothing
//   - SLP011: New test that only has assertion calls with no arrange
type SLP011 struct{}

func (SLP011) ID() string                { return "SLP011" }
func (SLP011) DefaultSeverity() Severity { return SeverityWarn }
func (SLP011) Description() string {
	return "test function body is only assertions with no arrange"
}

// variableAssignPattern matches a single variable assignment at statement
// level (not inside a nested block). This is the "arrange" pattern that
// is allowed before assertions.
// e.g. "got := Foo()" or "result := DoSomething()"
var variableAssignPattern = regexp.MustCompile(`^\s*(\w+)\s*:=\s*.+\s*$`)

// isSingleVariableAssign reports whether the line is a single variable
// assignment at statement level (top-level in the function body).
func isSingleVariableAssign(line string) bool {
	trimmed := strings.TrimSpace(line)
	return variableAssignPattern.MatchString(trimmed)
}

func (r SLP011) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		lang := testFileLang(f.Path)
		if lang == "" {
			continue
		}
		// SLP011 currently only handles Go test functions.
		// Non-Go languages don't have the same function-span detection
		// needed for assert-only test body analysis.
		if lang != "go" {
			continue
		}
		for _, h := range f.Hunks {
			findings := scanHunkForAssertOnlyTests(f.Path, h, r.DefaultSeverity(), r.ID())
			out = append(out, findings...)
		}
	}
	return out
}

// scanHunkForAssertOnlyTests walks a single hunk looking for added test
// function bodies that contain ONLY assertion calls (no meaningful logic).
// One variable assignment at the top is allowed as the "arrange" pattern.
//
// Key distinction:
//   - Top-level assertion (depth 1): assert.Equal(t, ...) → slop if no other logic
//   - Assertion inside if/for/etc at depth 1: if got := Foo(); assert.Equal(...) → NOT slop
//     (the if statement is meaningful logic, even if it contains an assertion)
//
// The rule fires when:
//   - All depth-1 statements are either assertions, single var assignments, or closing braces
//   - At least one depth-1 statement is a pure assertion (not inside control flow)
//   - AND no var assignment came before that pure assertion (the arrange was missing)
func scanHunkForAssertOnlyTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
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

		funcName, tVar := m[1], m[2]
		if isSafetyTestName(funcName) {
			i++
			continue
		}

		// Initial depth after signature line (the opening brace is on the same line).
		depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
		startLine := ln.NewLineNo

		// Track what we see at depth 1 (top-level function body).
		// sawTopLevelPureAssertion: saw an assertion statement at depth 1
		// sawTopLevelControlFlow: saw an if/for/etc statement at depth 1
		// sawTopLevelOther: saw any other meaningful statement at depth 1
		// sawVarAssignBeforeAssertion: saw a var assignment before any pure assertion
		sawTopLevelPureAssertion := false
		sawTopLevelControlFlow := false
		sawTopLevelOther := false
		sawVarAssignBeforeAssertion := false

		j := i + 1
		bodyAllAdded := true
		for j < len(lines) && depth > 0 {
			bl := lines[j]
			if bl.Kind != diff.LineAdd {
				// Non-added line inside function body - disqualify.
				bodyAllAdded = false
				break
			}

			// Update depth BEFORE processing the line content.
			depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")

			// Only consider lines at depth 1 for SLP011's logic.
			// Lines inside nested blocks (depth > 1) are not our concern here.
			if depth == 1 {
				trimmed := strings.TrimSpace(bl.Content)

				// Closing brace - nothing to check.
				if trimmed == "}" || trimmed == "" {
					// Continue to next line.
				} else if hasAssertion(trimmed, tVar) {
					// Check if this is a control flow statement containing an assertion.
					// e.g. "if got := Foo(); assert.Equal(...)" - the if is control flow.
					leading := trimmed
					if idx := strings.Index(leading, "//"); idx >= 0 {
						leading = strings.TrimSpace(leading[:idx])
					}
					if strings.HasPrefix(leading, "if ") ||
						strings.HasPrefix(leading, "for ") ||
						strings.HasPrefix(leading, "switch ") ||
						strings.HasPrefix(leading, "select ") {
						sawTopLevelControlFlow = true
					} else {
						sawTopLevelPureAssertion = true
					}
				} else if isSingleVariableAssign(trimmed) {
					// Single var assignment at top level is the "arrange" pattern.
					// Track that we saw it before any pure assertion.
					if !sawTopLevelPureAssertion {
						sawVarAssignBeforeAssertion = true
					}
				} else {
					// Any other statement at top level is meaningful logic.
					sawTopLevelOther = true
				}
			}

			j++
		}

		// Fire if:
		// - bodyAllAdded: entire function body was added in this diff
		// - depth == 0: we completed scanning the function (closed properly)
		// - !sawTopLevelOther: no meaningful non-assertion, non-arrange logic
		// - sawTopLevelPureAssertion: at least one pure assertion at top level
		// - !sawTopLevelControlFlow: no control flow statement (which would indicate logic)
		// - !sawVarAssignBeforeAssertion: no arrange was present before the first assertion
		if bodyAllAdded && depth == 0 && !sawTopLevelOther && sawTopLevelPureAssertion && !sawTopLevelControlFlow && !sawVarAssignBeforeAssertion {
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     startLine,
				Message:  "test function " + funcName + " body is only assertions — add arrange/act logic or delete it",
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}

		// Advance past the function body.
		if j > i {
			i = j
		} else {
			i++
		}
	}
	return out
}
