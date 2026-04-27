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

var slp092DoubleUnwrap = regexp.MustCompile(`(?i)(?:\.data){2,}\b|res\.data\.data\b|response\.data\.data\b`)

var slp092NoEnvelopeMock = regexp.MustCompile(`(?i)mock(?:Implementation|ReturnValue|ResolvedValue)(?:Once)?\s*\(\s*(?:(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*\(?\s*)?\{`)

var slp092EnvelopeKey = regexp.MustCompile(`(?i)\b(?:ok|status|success)\b\s*(?::|,|\})`)

var slp092EnvelopeDestructure = regexp.MustCompile(`(?i)(?:const|let|var)\s*\{[^}]*\b(?:ok|status|success)\b[^}]*\}\s*=.*await`)

// slp092HasEnvelopeInBlock checks whether the mock block starting at lines[start]
// contains an envelope key (ok/status/success). It counts brace depth from the
// opening line and scans forward until the block closes.
func slp092HasEnvelopeInBlock(lines []diff.Line, start int) bool {
	if start >= len(lines) {
		return false
	}
	braceDepth := strings.Count(lines[start].Content, "{") - strings.Count(lines[start].Content, "}")
	envelopeFound := slp092EnvelopeKey.MatchString(lines[start].Content)
	for j := start + 1; j < len(lines) && !envelopeFound; j++ {
		if braceDepth <= 0 {
			break
		}
		braceDepth += strings.Count(lines[j].Content, "{") - strings.Count(lines[j].Content, "}")
		if braceDepth == 1 && slp092EnvelopeKey.MatchString(lines[j].Content) {
			envelopeFound = true
		}
	}
	return envelopeFound
}

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

		lines := f.AddedLines()
		for i, ln := range lines {
			if slp092NoEnvelopeMock.MatchString(ln.Content) {
				if !slp092HasEnvelopeInBlock(lines, i) {
					hasNoEnvelopeMock = true
				}
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
			for i, ln := range lines {
				if !slp092NoEnvelopeMock.MatchString(ln.Content) {
					continue
				}
				if !slp092HasEnvelopeInBlock(lines, i) {
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
