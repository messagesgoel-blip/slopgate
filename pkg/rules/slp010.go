package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP010 flags pre-existing test functions where the ADDED lines
// contain no assertion. Unlike SLP001 (which catches entirely-new test
// functions with no assertions), SLP010 handles the incremental case:
// the AI edited an existing test and added setup/arrange code without
// adding a corresponding assertion.
//
// Languages: Go (full function-span tracking), JS/TS, Python, Java, Rust
// (simpler per-hunk assertion check).
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
		if f.IsDelete {
			continue
		}
		lang := testFileLang(f.Path)
		if lang == "" {
			continue
		}
		for _, h := range f.Hunks {
			var findings []Finding
			if lang == "go" {
				findings = scanHunkForIncrementalGoTests(f.Path, h, r.DefaultSeverity(), r.ID())
			} else {
				findings = scanHunkForIncrementalNonGoTests(f.Path, h, r.DefaultSeverity(), r.ID(), lang)
			}
			out = append(out, findings...)
		}
	}
	return out
}

// scanHunkForIncrementalGoTests walks a single hunk looking for ADDED lines
// inside EXISTING Go test functions (functions whose signature is NOT on an
// added line). If none of the added lines inside the test body contain an
// assertion, it emits a finding.
func scanHunkForIncrementalGoTests(path string, h diff.Hunk, sev Severity, ruleID string) []Finding {
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

// scanHunkForIncrementalNonGoTests checks if added lines in test files
// (JS/TS/Python/Java/Rust) contain any assertion tokens. For non-Go
// languages, we use a simpler approach: check all added lines in the
// test file for assertion presence.
func scanHunkForIncrementalNonGoTests(path string, h diff.Hunk, sev Severity, ruleID string, lang string) []Finding {
	var out []Finding
	// Skip pure new-file additions: SLP010 only fires when existing test
	// code is edited without adding assertions. Brand-new files are SLP001 territory.
	if h.OldStart == 0 && h.OldLines == 0 {
		return out
	}
	var addedLines []diff.Line
	for _, ln := range h.Lines {
		if ln.Kind == diff.LineAdd {
			addedLines = append(addedLines, ln)
		}
	}
	if len(addedLines) == 0 {
		return out
	}

	// Check if any added line contains an assertion for this language.
	sawAssertion := false
	for _, ln := range addedLines {
		if langHasAssertion(ln.Content, lang) {
			sawAssertion = true
			break
		}
	}

	if !sawAssertion {
		out = append(out, Finding{
			RuleID:   ruleID,
			Severity: sev,
			File:     path,
			Line:     addedLines[0].NewLineNo,
			Message:  "added lines in test file contain no assertion -- add an assertion or remove them",
			Snippet:  strings.TrimSpace(addedLines[0].Content),
		})
	}

	return out
}

// langHasAssertion reports whether a line contains an assertion token
// for the given language.
func langHasAssertion(line string, lang string) bool {
	switch lang {
	case "js":
		return jsHasAssertion(line)
	case "py":
		return pyHasAssertion(line)
	case "java":
		return javaHasAssertion(line)
	case "rust":
		return rustHasAssertion(line)
	}
	return false
}
