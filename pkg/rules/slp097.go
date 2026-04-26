package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP097 flags response destructuring patterns that may not match the
// API envelope. Common AI slop: frontend destructures { data } from
// response but the API wraps everything in { ok: true, data: { ... } },
// requiring res.data.data.X instead of res.data.X.
type SLP097 struct{}

func (SLP097) ID() string                { return "SLP097" }
func (SLP097) DefaultSeverity() Severity { return SeverityWarn }
func (SLP097) Description() string {
	return "response destructuring may not match API envelope — verify {ok, data} contract"
}

var slp097DestructureData = regexp.MustCompile(`(?i)(?:const|let|var)\s*\{[^}]*\bdata\b[^}]*\}\s*=\s*(?:await\s+)?\w+(?:\.\w+\([^)]*\))+`)

var slp097NoOkCheck = regexp.MustCompile(`(?i)fetch\(.+?\)\s*\.then\s*\(\s*\(?\s*([A-Za-z_$][\w$]*)(?:\s*:\s*[^)\r\n]+)?\s*\)?\s*=>\s*([A-Za-z_$][\w$]*)\s*\.json\s*\(\s*\)`)

func slp097MatchesNoOkCheck(content string) bool {
	match := slp097NoOkCheck.FindStringSubmatch(content)
	return len(match) == 3 && match[1] == match[2]
}

func (r SLP097) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)

			if slp097DestructureData.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "destructures data from response — verify API envelope ({ok, data}) and check for double-wrap",
					Snippet:  content,
				})
			}

			if slp097MatchesNoOkCheck(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "fetch without checking res.ok before .json() — 4xx/5xx response bodies will be treated as success",
					Snippet:  content,
				})
			}
		}
	}
	return out
}
