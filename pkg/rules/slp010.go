package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP010 flags pre-existing Go test functions where the ADDED lines
// contain no assertion. Unlike SLP001 (which catches entirely-new test
// functions with no assertions), SLP010 handles the incremental case:
// the AI edited an existing test and added setup/arrange code without
// adding a corresponding assertion.
//
// Example: an existing TestFoo gets a new line `result := Foo()` added,
// but no line checks the result. The test still compiles, coverage goes
// up, but nothing new is actually verified.
type SLP010 struct{}

func (SLP010) ID() string                { return "SLP010" }
func (SLP010) DefaultSeverity() Severity { return SeverityWarn }
func (SLP010) Description() string {
	return "added lines in existing test contain no assertion"
}

func (r SLP010) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		for _, h := range f.Hunks {
			findings := scanHunkForIncrementalTests(f.Path, h, r.DefaultSeverity(), r.ID())
			out = append(out, findings...)
		}
	}
	return out
}

// scanHunkForIncrementalTests walks a single hunk looking for ADDED lines
// inside EXISTING Go test functions (functions whose signature is NOT on an
// added line). If none of the added lines inside the test body contain an
// assertion, it emits a finding.
func scanHunkForIncrementalTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
	var out []Finding
	lines := h.Lines

	// Phase 1: identify all test function spans (name, tVar, start line index, end line index)
	// by tracking brace depth across all lines in the hunk.
	type testSpan struct {
		name     string
		tVar     string
		startIdx int // index in lines where func signature is
		endIdx   int // index in lines where closing brace is
		sigAdded bool
	}

	var spans []testSpan
	var currentSpan *testSpan
	depth := 0

	for i, ln := range lines {
		content := strings.TrimLeft(ln.Content, " \t")
		// Detect test function signature at depth 0.
		if depth == 0 {
			m := testFuncSignature.FindStringSubmatch(content)
			if m != nil {
				currentSpan = &testSpan{
					name:     m[1],
					tVar:     m[2],
					startIdx: i,
					sigAdded: ln.Kind == diff.LineAdd,
				}
				depth += strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
				continue
			}
		}
		if currentSpan != nil {
			depth += strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
			if depth <= 0 {
				currentSpan.endIdx = i
				spans = append(spans, *currentSpan)
				currentSpan = nil
				depth = 0
			}
		}
	}

	// Phase 2: for each existing test (sigAdded == false), check if any
	// added line inside the body has an assertion.
	for _, sp := range spans {
		if sp.sigAdded {
			// Entirely new test — SLP001's territory.
			continue
		}
		if isSafetyTestName(sp.name) {
			continue
		}

		hasAddedLines := false
		sawAssertion := false
		sawTopLevelSkip := false

		// Walk lines inside the function body (between signature and closing brace).
		// We start at depth 1 because the opening brace on the signature line put us
		// inside the function body already.
		bodyDepth := 1
		for i := sp.startIdx + 1; i <= sp.endIdx; i++ {
			ln := lines[i]
			if ln.Kind == diff.LineAdd {
				hasAddedLines = true
				if hasAssertion(ln.Content, sp.tVar) {
					sawAssertion = true
				}
				// A top-level skip at depth 1 (directly inside the function body)
				// marks the test as an intentional scaffold.
				if bodyDepth == 1 && isTopLevelSkipStatement(ln.Content, sp.tVar) {
					sawTopLevelSkip = true
				}
			}
			bodyDepth += strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
		}

		if hasAddedLines && !sawAssertion && !sawTopLevelSkip {
			// Find the first added line for the finding location.
			var firstAdded diff.Line
			for i := sp.startIdx + 1; i <= sp.endIdx; i++ {
				if lines[i].Kind == diff.LineAdd {
					firstAdded = lines[i]
					break
				}
			}
			out = append(out, Finding{
				RuleID:   ruleID,
				Severity: sev,
				File:     path,
				Line:     firstAdded.NewLineNo,
				Message:  "test function " + sp.name + " has added lines with no assertion — add t.Error/t.Fatal/assert.* or delete them",
				Snippet:  strings.TrimSpace(firstAdded.Content),
			})
		}
	}

	return out
}
