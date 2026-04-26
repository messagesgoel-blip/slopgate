package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP099 detects when a response struct/type field is added, renamed, or
// retyped in a non-test file without corresponding test file changes in
// the same diff. This is a common AI slop pattern: the agent changes a
// response shape but doesn't update the tests, causing test drift.
type SLP099 struct{}

func (SLP099) ID() string                { return "SLP099" }
func (SLP099) DefaultSeverity() Severity { return SeverityWarn }
func (SLP099) Description() string {
	return "response field changed without test update — tests may be stale"
}

var slp099GoStructField = regexp.MustCompile(`^\s*\w+\s+(?:\[\])?\*?\w+(?:\.\w+)?\s+\x60[^\x60]*\x60`)

var slp099TSInterfaceProp = regexp.MustCompile(`(?i)(?:readonly\s+)?\w+(?:\?)?:\s*(?:string|number|boolean|Date|\[\]\w+|\w+\[\])[;,]?$`)

var slp099ResponseKeywords = []string{"Response", "response", "Res", "res", "DTO", "dto", "Output", "output", "Result", "result", "Payload"}

func hasResponseKeyword(name string) bool {
	for _, kw := range slp099ResponseKeywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

func (r SLP099) Check(d *diff.Diff) []Finding {
	var out []Finding
	changedFiles := make(map[string]bool)
	changedTestFiles := make(map[string]bool)

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			changedTestFiles[f.Path] = true
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if slp099GoStructField.MatchString(content) || slp099TSInterfaceProp.MatchString(content) {
				if hasResponseKeyword(f.Path) {
					changedFiles[f.Path] = true
				}
			}
		}
	}

	for _, f := range d.Files {
		if !changedFiles[f.Path] {
			continue
		}
		if testMatchesResponse(f.Path, changedTestFiles) {
			continue
		}
		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if slp099GoStructField.MatchString(content) || slp099TSInterfaceProp.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "response field added/changed without test update — verify tests still match",
					Snippet:  content,
				})
			}
		}
	}
	return out
}

func testMatchesResponse(respPath string, testFiles map[string]bool) bool {
	if len(testFiles) == 0 {
		return false
	}
	// derive basename and stem for the response file
	base := respPath
	if i := strings.LastIndex(respPath, "/"); i >= 0 {
		base = respPath[i+1:]
	}
	stem := base
	if i := strings.LastIndex(base, "."); i >= 0 {
		stem = base[:i]
	}
	// check if any test file contains the same stem
	for tf := range testFiles {
		if strings.Contains(tf, stem) {
			return true
		}
	}
	return false
}
