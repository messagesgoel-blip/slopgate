package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP092 detects mock return values that don't match the API envelope shape
// expected by the consuming code. A common AI slop pattern is mocking an API
// response as { data: ... } when the actual API wraps in { ok: true, data: ... },
// or vice versa.
//
// Heuristic: find mockResolvedValue/mockReturnValue calls, inspect the shape,
// compare against how the response is destructured in the same hunk or file.
type SLP092 struct{}

func (SLP092) ID() string                { return "SLP092" }
func (SLP092) DefaultSeverity() Severity { return SeverityWarn }
func (SLP092) Description() string {
	return "mock return shape may not match API envelope — verify against actual contract"
}

var slp092MockReturn = regexp.MustCompile(`(?i)mock(?:Resolved|Return|Implementation|Resolved)Value(?:Once)?\s*\(\s*(\{[^}]*\}|[^)]+)\)`)

var slp092DoubleUnwrap = regexp.MustCompile(`(?i)(?:\.data){2,}\b|res\.data\.data\b|response\.data\.data\b`)

var slp092NoEnvelopeMock = regexp.MustCompile(`mock\w*Value\w*\s*\(\s*\{`)

var slp092EnvelopeDestructure = regexp.MustCompile(`(?i)(?:const|let|var)\s*\{[^}]*\b(?:ok|status|success)\b[^}]*\}\s*=.*await`)

func (r SLP092) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isTestFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) {
			continue
		}

		hasNoEnvelopeMock := false
		hasEnvelopeDestructure := false

		for _, ln := range f.AddedLines() {
			if slp092NoEnvelopeMock.MatchString(ln.Content) {
				hasNoEnvelopeMock = true
			}
			if slp092EnvelopeDestructure.MatchString(ln.Content) {
				hasEnvelopeDestructure = true
			}
			if slp092DoubleUnwrap.MatchString(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "response double-unwrapped — check if mock shape matches API envelope",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}

		if hasNoEnvelopeMock && hasEnvelopeDestructure {
			for _, ln := range f.AddedLines() {
				if slp092NoEnvelopeMock.MatchString(ln.Content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "mock returns object without envelope but code expects {ok, data} pattern",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}
